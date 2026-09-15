package auth

import (
	"testing"
	"time"
)

func TestTOTPMatchesRFC6238Vector(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	for _, test := range []struct {
		at   int64
		want string
	}{
		{at: 59, want: "287082"},
		{at: 1111111109, want: "081804"},
		{at: 1111111111, want: "050471"},
	} {
		got, err := TOTPCode(secret, time.Unix(test.at, 0).UTC())
		if err != nil || got != test.want {
			t.Fatalf("TOTP at %d = %q, err=%v, want %q", test.at, got, err, test.want)
		}
	}
	if VerifyTOTP(secret, "287082", time.Unix(59, 0).UTC()) == false {
		t.Fatal("valid TOTP was rejected")
	}
	if VerifyTOTP(secret, "000000", time.Unix(59, 0).UTC()) {
		t.Fatal("invalid TOTP was accepted")
	}
}

func TestMFASecretEncryptionAndRecoveryCodeBounds(t *testing.T) {
	key := []byte("01234567890123456789012345678901")
	ciphertext, err := EncryptSecret(key, "TOPSECRETBASE32")
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := DecryptSecret(key, ciphertext)
	if err != nil || plaintext != "TOPSECRETBASE32" {
		t.Fatalf("MFA secret roundtrip = %q, err=%v", plaintext, err)
	}
	if _, err := DecryptSecret([]byte("wrong-key-wrong-key-wrong-key-123"), ciphertext); err == nil {
		t.Fatal("MFA secret decrypted with wrong key")
	}
	codes, err := GenerateRecoveryCodes(10)
	if err != nil || len(codes) != 10 {
		t.Fatalf("recovery codes = %d, err=%v", len(codes), err)
	}
	if NormalizeRecoveryCode(codes[0]) == codes[0] {
		t.Fatal("recovery normalization did not remove separators")
	}
	if _, err := GenerateRecoveryCodes(11); err == nil {
		t.Fatal("recovery code bound was not enforced")
	}
}
