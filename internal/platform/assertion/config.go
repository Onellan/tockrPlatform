package assertion

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

// ConfigFromEnvironment keeps private key parsing at the startup boundary.
// Secret material is never included in returned errors.
func ConfigFromEnvironment(getenv func(string) string) (Config, error) {
	if getenv == nil {
		return Config{}, fmt.Errorf("%w: environment reader", ErrConfiguration)
	}
	issuer := strings.TrimSpace(getenv("PLATFORM_ASSERTION_ISSUER"))
	keyID := strings.TrimSpace(getenv("PLATFORM_ASSERTION_KEY_ID"))
	privateText := strings.TrimSpace(getenv("PLATFORM_ASSERTION_PRIVATE_KEY"))
	audiencesText := strings.TrimSpace(getenv("PLATFORM_ASSERTION_AUDIENCES"))
	if issuer == "" || keyID == "" || privateText == "" || audiencesText == "" {
		return Config{}, fmt.Errorf("%w: issuer, key id, private key and audiences are required", ErrConfiguration)
	}
	privateKey, err := decodeKey(privateText, ed25519.PrivateKeySize)
	if err != nil {
		return Config{}, fmt.Errorf("%w: private key encoding", ErrConfiguration)
	}
	audiences := splitValues(audiencesText)
	if len(audiences) == 0 {
		return Config{}, fmt.Errorf("%w: audiences", ErrConfiguration)
	}
	verificationKeys := make(map[string]ed25519.PublicKey)
	for _, item := range splitValues(getenv("PLATFORM_ASSERTION_VERIFY_KEYS")) {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return Config{}, fmt.Errorf("%w: verification key entry", ErrConfiguration)
		}
		keyID := strings.TrimSpace(parts[0])
		if _, exists := verificationKeys[keyID]; exists {
			return Config{}, fmt.Errorf("%w: duplicate verification key", ErrConfiguration)
		}
		publicKey, decodeErr := decodeKey(strings.TrimSpace(parts[1]), ed25519.PublicKeySize)
		if decodeErr != nil {
			return Config{}, fmt.Errorf("%w: verification key encoding", ErrConfiguration)
		}
		verificationKeys[keyID] = ed25519.PublicKey(publicKey)
	}
	return Config{Issuer: issuer, ActiveKeyID: keyID, ActivePrivateKey: ed25519.PrivateKey(privateKey), VerificationKeys: verificationKeys, Audiences: audiences}, nil
}

func splitValues(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func decodeKey(value string, size int) ([]byte, error) {
	if decoded, err := hex.DecodeString(value); err == nil && len(decoded) == size {
		return decoded, nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(value); err == nil && len(decoded) == size {
		return decoded, nil
	}
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil && len(decoded) == size {
		return decoded, nil
	}
	return nil, fmt.Errorf("unsupported key encoding")
}
