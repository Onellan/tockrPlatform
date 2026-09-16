// Package assertion owns the Platform-signed, short-lived handoff contract.
// It deliberately contains no product-role, billing or product-database
// authority.
package assertion

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	Algorithm       = "Ed25519"
	Type            = "TockrPlatformAssertion"
	CurrentVersion  = 1
	DefaultLifetime = 2 * time.Minute
)

var (
	ErrConfiguration        = errors.New("invalid assertion configuration")
	ErrInvalidAssertion     = errors.New("invalid platform assertion")
	ErrInvalidAudience      = errors.New("invalid assertion audience")
	ErrInvalidScope         = errors.New("invalid assertion scope")
	ErrAssertionExpired     = errors.New("platform assertion expired")
	ErrAssertionNotYetValid = errors.New("platform assertion is not yet valid")
	ErrAssertionReplay      = errors.New("platform assertion replay detected")
)

// Claims is the complete v1 assertion payload. Keep this allow-list narrow:
// consumers must obtain product roles and billing decisions from their own
// product authority.
type Claims struct {
	Issuer                 string    `json:"issuer"`
	Audience               string    `json:"audience"`
	PlatformUserID         string    `json:"platform_user_id"`
	PlatformOrganisationID string    `json:"platform_organisation_id"`
	PlatformWorkspaceID    string    `json:"platform_workspace_id"`
	IssuedAt               time.Time `json:"issued_at"`
	ExpiresAt              time.Time `json:"expires_at"`
	AssertionID            string    `json:"assertion_id"`
	AssertionVersion       int       `json:"assertion_version"`
}

type header struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
	Type      string `json:"typ"`
}

type IssueRequest struct {
	Audience       string
	PlatformUserID string
	OrganisationID string
	WorkspaceID    string
	Lifetime       time.Duration
}

type Config struct {
	Issuer           string
	ActiveKeyID      string
	ActivePrivateKey ed25519.PrivateKey
	VerificationKeys map[string]ed25519.PublicKey
	Audiences        []string
	MaxLifetime      time.Duration
	ClockSkew        time.Duration
}

type VerifierConfig struct {
	Issuer           string
	VerificationKeys map[string]ed25519.PublicKey
	Audiences        []string
	MaxLifetime      time.Duration
	ClockSkew        time.Duration
}

type VerificationKey struct {
	KeyID     string
	Algorithm string
	PublicKey ed25519.PublicKey
}

type Issuer struct {
	issuer      string
	activeKeyID string
	privateKey  ed25519.PrivateKey
	publicKeys  map[string]ed25519.PublicKey
	audiences   map[string]struct{}
	maxLifetime time.Duration
	clockSkew   time.Duration

	replayMu sync.Mutex
	replayed map[string]time.Time
}

func New(config Config) (*Issuer, error) {
	issuer := strings.TrimSpace(config.Issuer)
	if !validText(issuer, 200) || strings.ContainsAny(issuer, "\r\n") {
		return nil, fmt.Errorf("%w: issuer", ErrConfiguration)
	}
	activeKeyID := strings.TrimSpace(config.ActiveKeyID)
	if !validText(activeKeyID, 100) {
		return nil, fmt.Errorf("%w: active key id", ErrConfiguration)
	}
	if len(config.ActivePrivateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("%w: active private key", ErrConfiguration)
	}
	maxLifetime := config.MaxLifetime
	if maxLifetime == 0 {
		maxLifetime = 5 * time.Minute
	}
	if maxLifetime < DefaultLifetime || maxLifetime > 15*time.Minute {
		return nil, fmt.Errorf("%w: max lifetime", ErrConfiguration)
	}
	clockSkew := config.ClockSkew
	if clockSkew == 0 {
		clockSkew = 5 * time.Second
	}
	if clockSkew < 0 || clockSkew > time.Minute {
		return nil, fmt.Errorf("%w: clock skew", ErrConfiguration)
	}
	if len(config.Audiences) == 0 {
		return nil, fmt.Errorf("%w: audiences", ErrConfiguration)
	}
	audiences := make(map[string]struct{}, len(config.Audiences))
	for _, value := range config.Audiences {
		value = strings.TrimSpace(value)
		if !validText(value, 100) || strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("%w: audience", ErrConfiguration)
		}
		if _, exists := audiences[value]; exists {
			return nil, fmt.Errorf("%w: duplicate audience", ErrConfiguration)
		}
		audiences[value] = struct{}{}
	}
	activePublic, ok := config.ActivePrivateKey.Public().(ed25519.PublicKey)
	if !ok || len(activePublic) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%w: active public key", ErrConfiguration)
	}
	publicKeys := make(map[string]ed25519.PublicKey, len(config.VerificationKeys)+1)
	for keyID, publicKey := range config.VerificationKeys {
		if !validText(keyID, 100) || len(publicKey) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("%w: verification key", ErrConfiguration)
		}
		publicKeys[keyID] = append(ed25519.PublicKey(nil), publicKey...)
	}
	if existing, exists := publicKeys[activeKeyID]; exists && subtle.ConstantTimeCompare(existing, activePublic) != 1 {
		return nil, fmt.Errorf("%w: active public key mismatch", ErrConfiguration)
	}
	publicKeys[activeKeyID] = append(ed25519.PublicKey(nil), activePublic...)
	return &Issuer{
		issuer:      issuer,
		activeKeyID: activeKeyID,
		privateKey:  append(ed25519.PrivateKey(nil), config.ActivePrivateKey...),
		publicKeys:  publicKeys,
		audiences:   audiences,
		maxLifetime: maxLifetime,
		clockSkew:   clockSkew,
		replayed:    make(map[string]time.Time),
	}, nil
}

// NewVerifier constructs the consumer-side boundary with public verification
// keys only. Consumers never need or receive Platform private signing
// material.
func NewVerifier(config VerifierConfig) (*Issuer, error) {
	issuer := strings.TrimSpace(config.Issuer)
	if !validText(issuer, 200) || strings.ContainsAny(issuer, "\r\n") {
		return nil, fmt.Errorf("%w: issuer", ErrConfiguration)
	}
	maxLifetime := config.MaxLifetime
	if maxLifetime == 0 {
		maxLifetime = 5 * time.Minute
	}
	if maxLifetime < DefaultLifetime || maxLifetime > 15*time.Minute {
		return nil, fmt.Errorf("%w: max lifetime", ErrConfiguration)
	}
	clockSkew := config.ClockSkew
	if clockSkew == 0 {
		clockSkew = 5 * time.Second
	}
	if clockSkew < 0 || clockSkew > time.Minute {
		return nil, fmt.Errorf("%w: clock skew", ErrConfiguration)
	}
	if len(config.Audiences) == 0 || len(config.VerificationKeys) == 0 {
		return nil, fmt.Errorf("%w: audiences and verification keys", ErrConfiguration)
	}
	audiences := make(map[string]struct{}, len(config.Audiences))
	for _, value := range config.Audiences {
		value = strings.TrimSpace(value)
		if !validText(value, 100) || strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("%w: audience", ErrConfiguration)
		}
		if _, exists := audiences[value]; exists {
			return nil, fmt.Errorf("%w: duplicate audience", ErrConfiguration)
		}
		audiences[value] = struct{}{}
	}
	publicKeys := make(map[string]ed25519.PublicKey, len(config.VerificationKeys))
	for keyID, publicKey := range config.VerificationKeys {
		keyID = strings.TrimSpace(keyID)
		if !validText(keyID, 100) || len(publicKey) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("%w: verification key", ErrConfiguration)
		}
		publicKeys[keyID] = append(ed25519.PublicKey(nil), publicKey...)
	}
	return &Issuer{
		issuer:      issuer,
		publicKeys:  publicKeys,
		audiences:   audiences,
		maxLifetime: maxLifetime,
		clockSkew:   clockSkew,
		replayed:    make(map[string]time.Time),
	}, nil
}

func (i *Issuer) Issue(request IssueRequest, at time.Time) (string, Claims, error) {
	if i == nil {
		return "", Claims{}, ErrConfiguration
	}
	if len(i.privateKey) != ed25519.PrivateKeySize {
		return "", Claims{}, ErrConfiguration
	}
	audience := strings.TrimSpace(request.Audience)
	if _, ok := i.audiences[audience]; !ok {
		return "", Claims{}, ErrInvalidAudience
	}
	if !validScopeID(request.PlatformUserID, "usr_") || !validScopeID(request.OrganisationID, "org_") || !validScopeID(request.WorkspaceID, "wsp_") {
		return "", Claims{}, ErrInvalidScope
	}
	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}
	lifetime := request.Lifetime
	if lifetime == 0 {
		lifetime = DefaultLifetime
	}
	if lifetime <= 0 || lifetime > i.maxLifetime {
		return "", Claims{}, fmt.Errorf("%w: assertion lifetime", ErrInvalidAssertion)
	}
	assertionID, err := newAssertionID()
	if err != nil {
		return "", Claims{}, err
	}
	claims := Claims{
		Issuer:                 i.issuer,
		Audience:               audience,
		PlatformUserID:         request.PlatformUserID,
		PlatformOrganisationID: request.OrganisationID,
		PlatformWorkspaceID:    request.WorkspaceID,
		IssuedAt:               at,
		ExpiresAt:              at.Add(lifetime),
		AssertionID:            assertionID,
		AssertionVersion:       CurrentVersion,
	}
	return i.sign(claims)
}

func (i *Issuer) Verify(token string, now time.Time) (Claims, error) {
	if i == nil {
		return Claims{}, ErrConfiguration
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 || len(parts[0]) > 2048 || len(parts[1]) > 4096 || len(parts[2]) > 4096 {
		return Claims{}, ErrInvalidAssertion
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrInvalidAssertion
	}
	var value header
	if err := decodeStrict(headerBytes, &value); err != nil || value.Algorithm != Algorithm || value.Type != Type || !validText(value.KeyID, 100) {
		return Claims{}, ErrInvalidAssertion
	}
	publicKey, ok := i.publicKeys[value.KeyID]
	if !ok {
		return Claims{}, ErrInvalidAssertion
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(signature) != ed25519.SignatureSize || !ed25519.Verify(publicKey, []byte(parts[0]+"."+parts[1]), signature) {
		return Claims{}, ErrInvalidAssertion
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidAssertion
	}
	var claims Claims
	if err := decodeStrict(payload, &claims); err != nil {
		return Claims{}, ErrInvalidAssertion
	}
	if err := i.validateClaims(claims, now); err != nil {
		return Claims{}, err
	}
	i.replayMu.Lock()
	for assertionID, expiresAt := range i.replayed {
		if !expiresAt.After(now.UTC()) {
			delete(i.replayed, assertionID)
		}
	}
	if _, exists := i.replayed[claims.AssertionID]; exists {
		i.replayMu.Unlock()
		return Claims{}, ErrAssertionReplay
	}
	i.replayed[claims.AssertionID] = claims.ExpiresAt
	i.replayMu.Unlock()
	return claims, nil
}

func (i *Issuer) PublicKeys() []VerificationKey {
	if i == nil {
		return nil
	}
	keys := make([]VerificationKey, 0, len(i.publicKeys))
	if publicKey, ok := i.publicKeys[i.activeKeyID]; ok {
		keys = append(keys, VerificationKey{KeyID: i.activeKeyID, Algorithm: Algorithm, PublicKey: append(ed25519.PublicKey(nil), publicKey...)})
	}
	for keyID, publicKey := range i.publicKeys {
		if keyID == i.activeKeyID {
			continue
		}
		keys = append(keys, VerificationKey{KeyID: keyID, Algorithm: Algorithm, PublicKey: append(ed25519.PublicKey(nil), publicKey...)})
	}
	if len(keys) > 1 {
		sort.Slice(keys[1:], func(left, right int) bool { return keys[left+1].KeyID < keys[right+1].KeyID })
	}
	return keys
}

func (i *Issuer) sign(claims Claims) (string, Claims, error) {
	headerBytes, err := json.Marshal(header{Algorithm: Algorithm, KeyID: i.activeKeyID, Type: Type})
	if err != nil {
		return "", Claims{}, fmt.Errorf("marshal assertion header: %w", err)
	}
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", Claims{}, fmt.Errorf("marshal assertion claims: %w", err)
	}
	headerEncoded := base64.RawURLEncoding.EncodeToString(headerBytes)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signed := headerEncoded + "." + payloadEncoded
	signature := ed25519.Sign(i.privateKey, []byte(signed))
	return signed + "." + base64.RawURLEncoding.EncodeToString(signature), claims, nil
}

func (i *Issuer) validateClaims(claims Claims, now time.Time) error {
	if claims.AssertionVersion != CurrentVersion {
		return ErrAssertionVersionMismatch
	}
	if claims.Issuer != i.issuer {
		return ErrInvalidAssertion
	}
	if _, ok := i.audiences[claims.Audience]; !ok {
		return ErrInvalidAudience
	}
	if !validScopeID(claims.PlatformUserID, "usr_") || !validScopeID(claims.PlatformOrganisationID, "org_") || !validScopeID(claims.PlatformWorkspaceID, "wsp_") || !validScopeID(claims.AssertionID, "ast_") {
		return ErrInvalidScope
	}
	if claims.IssuedAt.IsZero() || claims.ExpiresAt.IsZero() || !claims.ExpiresAt.After(claims.IssuedAt) || claims.ExpiresAt.Sub(claims.IssuedAt) > i.maxLifetime {
		return ErrInvalidAssertion
	}
	if claims.IssuedAt.After(now.Add(i.clockSkew)) {
		return ErrAssertionNotYetValid
	}
	if !claims.ExpiresAt.After(now) {
		return ErrAssertionExpired
	}
	return nil
}

func decodeStrict(data []byte, target any) error {
	if err := rejectDuplicateObjectKeys(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("trailing assertion data")
	}
	return nil
}

func rejectDuplicateObjectKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok || delimiter != '{' {
		return errors.New("assertion object required")
	}
	seen := make(map[string]struct{})
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := keyToken.(string)
		if !ok {
			return errors.New("assertion object key required")
		}
		if _, exists := seen[key]; exists {
			return errors.New("duplicate assertion object key")
		}
		seen[key] = struct{}{}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return err
		}
	}
	if token, err := decoder.Token(); err != nil || token != json.Delim('}') {
		return errors.New("assertion object not closed")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("trailing assertion data")
	}
	return nil
}

func newAssertionID() (string, error) {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate assertion id: %w", err)
	}
	return "ast_" + base64.RawURLEncoding.EncodeToString(raw), nil
}

func validScopeID(value, prefix string) bool {
	return strings.HasPrefix(value, prefix) && len(value) > len(prefix) && len(value) <= 200 && !strings.ContainsAny(value, "\r\n \t")
}

func validText(value string, max int) bool {
	value = strings.TrimSpace(value)
	return value != "" && len(value) <= max && !strings.ContainsAny(value, "\r\n\t ")
}
