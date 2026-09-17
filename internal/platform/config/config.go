// Package config owns the Platform process configuration boundary.
package config

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/Onellan/tockrplatform/internal/platform/readauthority"
)

const (
	defaultDBPath       = "platform.db"
	defaultHTTPAddr     = ":8080"
	defaultMaxHeaders   = 1 << 20
	defaultReadHeader   = 10 * time.Second
	defaultReadTimeout  = 15 * time.Second
	defaultWriteTimeout = 30 * time.Second
	defaultIdleTimeout  = 120 * time.Second
	defaultShutdown     = 10 * time.Second
)

var ErrConfiguration = errors.New("invalid Platform runtime configuration")

type Config struct {
	DBPath              string
	MFAKey              []byte
	HTTPAddr            string
	AllowInsecureCookie bool
	MaxHeaderBytes      int
	ReadHeaderTimeout   time.Duration
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	IdleTimeout         time.Duration
	ShutdownTimeout     time.Duration
	ReadAuthorityKeys   readauthority.PublicKeySet
}

// FromEnvironment validates process configuration without echoing secret
// material. Defaults are safe for local development and the container sets an
// explicit persistent database path.
func FromEnvironment(getenv func(string) string) (Config, error) {
	if getenv == nil {
		return Config{}, fmt.Errorf("%w: environment reader", ErrConfiguration)
	}

	dbPath := strings.TrimSpace(getenv("PLATFORM_DB_PATH"))
	if dbPath == "" {
		dbPath = defaultDBPath
	}
	if strings.IndexFunc(dbPath, func(r rune) bool { return r == '\x00' }) >= 0 {
		return Config{}, fmt.Errorf("%w: database path", ErrConfiguration)
	}

	keyText := strings.TrimSpace(getenv("PLATFORM_MFA_KEY"))
	key, err := hex.DecodeString(keyText)
	if err != nil || len(key) != 32 {
		return Config{}, fmt.Errorf("%w: PLATFORM_MFA_KEY must be configured as 64 hex characters", ErrConfiguration)
	}

	addr := strings.TrimSpace(getenv("PLATFORM_HTTP_ADDR"))
	if addr == "" {
		addr = defaultHTTPAddr
	}
	if err := validateHTTPAddr(addr); err != nil {
		return Config{}, fmt.Errorf("%w: PLATFORM_HTTP_ADDR", ErrConfiguration)
	}

	allowInsecure, err := parseBoolean(getenv("PLATFORM_ALLOW_INSECURE_COOKIES"))
	if err != nil {
		return Config{}, fmt.Errorf("%w: PLATFORM_ALLOW_INSECURE_COOKIES", ErrConfiguration)
	}
	readAuthorityKeys, err := parseReadAuthorityKeys(getenv("PLATFORM_READ_AUTHORITY_KEYS"))
	if err != nil {
		return Config{}, fmt.Errorf("%w: PLATFORM_READ_AUTHORITY_KEYS", ErrConfiguration)
	}

	return Config{
		DBPath:              dbPath,
		MFAKey:              append([]byte(nil), key...),
		HTTPAddr:            addr,
		AllowInsecureCookie: allowInsecure,
		MaxHeaderBytes:      defaultMaxHeaders,
		ReadHeaderTimeout:   defaultReadHeader,
		ReadTimeout:         defaultReadTimeout,
		WriteTimeout:        defaultWriteTimeout,
		IdleTimeout:         defaultIdleTimeout,
		ShutdownTimeout:     defaultShutdown,
		ReadAuthorityKeys:   readAuthorityKeys,
	}, nil
}

func parseReadAuthorityKeys(value string) (readauthority.PublicKeySet, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return readauthority.PublicKeySet{}, nil
	}
	var encoded map[string]map[string]string
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&encoded); err != nil {
		return nil, err
	}
	if encoded == nil {
		return nil, errors.New("read-authority key set must be an object")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, errors.New("read-authority key set contains trailing data")
	}
	keys := make(readauthority.PublicKeySet, len(encoded))
	for consumer, consumerKeys := range encoded {
		if err := readauthority.ValidateConsumer(consumer); err != nil || len(consumerKeys) == 0 {
			return nil, errors.New("invalid consumer key set")
		}
		keys[consumer] = make(map[string]ed25519.PublicKey, len(consumerKeys))
		for keyID, text := range consumerKeys {
			trimmedKeyID := strings.TrimSpace(keyID)
			if keyID != trimmedKeyID || !readauthority.ValidateKeyID(trimmedKeyID) {
				return nil, errors.New("invalid key ID")
			}
			decoded, err := hex.DecodeString(strings.TrimSpace(text))
			if err != nil || len(decoded) != ed25519.PublicKeySize {
				return nil, errors.New("invalid public key")
			}
			keys[consumer][trimmedKeyID] = append(ed25519.PublicKey(nil), decoded...)
		}
	}
	return keys, nil
}

func validateHTTPAddr(addr string) error {
	if strings.IndexFunc(addr, func(r rune) bool { return r == '\x00' || r == '\r' || r == '\n' || r == ' ' || r == '\t' }) >= 0 {
		return errors.New("address contains whitespace or control characters")
	}
	_, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return errors.New("address must contain a host and port")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("address port is invalid")
	}
	return nil
}

func parseBoolean(value string) (bool, error) {
	switch strings.TrimSpace(value) {
	case "", "0":
		return false, nil
	case "1":
		return true, nil
	default:
		return false, errors.New("boolean must be 0 or 1")
	}
}
