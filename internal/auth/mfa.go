package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" // #nosec G505 -- RFC 6238 TOTP requires HMAC-SHA1.
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrMFAKeyUnavailable = errors.New("MFA encryption key is unavailable")

func GenerateTOTPSecret() (string, error) {
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate MFA secret: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw), nil
}

func TOTPCode(secret string, at time.Time) (string, error) {
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil || len(decoded) == 0 {
		return "", errors.New("invalid MFA secret")
	}
	counter := uint64(at.UTC().Unix() / 30)
	var message [8]byte
	binary.BigEndian.PutUint64(message[:], counter)
	mac := hmac.New(sha1.New, decoded) // #nosec G505 -- required by RFC 6238.
	_, _ = mac.Write(message[:])
	digest := mac.Sum(nil)
	offset := digest[len(digest)-1] & 0x0f
	value := (uint32(digest[offset])&0x7f)<<24 |
		uint32(digest[offset+1])<<16 |
		uint32(digest[offset+2])<<8 |
		uint32(digest[offset+3])
	return fmt.Sprintf("%06d", value%1000000), nil
}

func VerifyTOTP(secret, code string, at time.Time) bool {
	_, valid := MatchTOTP(secret, code, at)
	return valid
}

func MatchTOTP(secret, code string, at time.Time) (int64, bool) {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return 0, false
	}
	baseStep := at.UTC().Unix() / 30
	for offset := -1; offset <= 1; offset++ {
		step := baseStep + int64(offset)
		candidate, err := TOTPCode(secret, time.Unix(step*30, 0).UTC())
		if err == nil && subtle.ConstantTimeCompare([]byte(candidate), []byte(code)) == 1 {
			return step, true
		}
	}
	return 0, false
}

func GenerateRecoveryCodes(count int) ([]string, error) {
	if count < 1 || count > 10 {
		return nil, errors.New("recovery code count must be between 1 and 10")
	}
	codes := make([]string, 0, count)
	for i := 0; i < count; i++ {
		raw := make([]byte, 8)
		if _, err := rand.Read(raw); err != nil {
			return nil, fmt.Errorf("generate recovery code: %w", err)
		}
		hexCode := strings.ToUpper(hex.EncodeToString(raw))
		codes = append(codes, hexCode[:4]+"-"+hexCode[4:8]+"-"+hexCode[8:12]+"-"+hexCode[12:])
	}
	return codes, nil
}

func NormalizeRecoveryCode(code string) string {
	return strings.ToUpper(strings.NewReplacer("-", "", " ", "", "\t", "").Replace(strings.TrimSpace(code)))
}

func EncryptSecret(key []byte, plaintext string) ([]byte, error) {
	block, err := newSecretBlock(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create MFA cipher: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate MFA nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

func DecryptSecret(key, ciphertext []byte) (string, error) {
	block, err := newSecretBlock(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create MFA cipher: %w", err)
	}
	if len(ciphertext) < gcm.NonceSize() {
		return "", errors.New("invalid encrypted MFA secret")
	}
	plaintext, err := gcm.Open(nil, ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():], nil)
	if err != nil {
		return "", errors.New("invalid encrypted MFA secret")
	}
	return string(plaintext), nil
}

func newSecretBlock(key []byte) (cipher.Block, error) {
	if len(key) != 32 {
		return nil, ErrMFAKeyUnavailable
	}
	return aes.NewCipher(key)
}
