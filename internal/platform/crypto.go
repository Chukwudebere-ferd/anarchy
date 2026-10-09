package platform

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
)

// LoadContactKey loads ANARCHY_CONTACT_KEY as 32 raw bytes.
// Accepts base64 (44 chars) or 64-char hex or 32-char raw. Generate with:
//   openssl rand -base64 32
func LoadContactKey() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv("ANARCHY_CONTACT_KEY"))
	if raw == "" {
		return nil, errors.New("platform: ANARCHY_CONTACT_KEY is not set")
	}
	// Try base64 first.
	if b, err := base64.StdEncoding.DecodeString(raw); err == nil && len(b) == 32 {
		return b, nil
	}
	if b, err := base64.RawStdEncoding.DecodeString(raw); err == nil && len(b) == 32 {
		return b, nil
	}
	// Raw 32 bytes passed directly.
	if len(raw) == 32 {
		return []byte(raw), nil
	}
	return nil, errors.New("platform: ANARCHY_CONTACT_KEY must decode to 32 bytes (openssl rand -base64 32)")
}

// Encrypt encrypts plaintext with AES-256-GCM. Returns nonce|ciphertext.
func Encrypt(key, plaintext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("platform: key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	out := gcm.Seal(nonce, nonce, plaintext, nil)
	return out, nil
}

// Decrypt reverses Encrypt.
func Decrypt(key, data []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("platform: key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(data) < gcm.NonceSize() {
		return nil, errors.New("platform: ciphertext too short")
	}
	nonce, ct := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("platform: decrypt failed: %w", err)
	}
	return plain, nil
}

func EncryptString(key []byte, s string) ([]byte, error) {
	if s == "" {
		return nil, nil
	}
	return Encrypt(key, []byte(s))
}

func DecryptString(key, data []byte) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	b, err := Decrypt(key, data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
