package platform

import (
	"bytes"
	"encoding/base64"
	"os"
	"testing"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	msg := []byte("sensitive: ferd@example.com")
	ct, err := Encrypt(key, msg)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ct, msg) {
		t.Fatal("ciphertext contains plaintext")
	}
	got, err := Decrypt(key, ct)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, msg) {
		t.Fatalf("roundtrip mismatch: %q", got)
	}
}

func TestDecryptTamperFails(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	ct, _ := Encrypt(key, []byte("hello"))
	ct[len(ct)-1] ^= 0xFF
	if _, err := Decrypt(key, ct); err == nil {
		t.Fatal("tampered ciphertext must not decrypt")
	}
}

func TestDecryptWrongKeyFails(t *testing.T) {
	ct, _ := Encrypt(bytes.Repeat([]byte{1}, 32), []byte("hello"))
	if _, err := Decrypt(bytes.Repeat([]byte{2}, 32), ct); err == nil {
		t.Fatal("wrong key must not decrypt")
	}
}

func TestLoadContactKeyBase64(t *testing.T) {
	raw := bytes.Repeat([]byte{9}, 32)
	t.Setenv("ANARCHY_CONTACT_KEY", base64.StdEncoding.EncodeToString(raw))
	got, err := LoadContactKey()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("decoded key mismatch: %v", got)
	}
}

func TestLoadContactKeyRejectsBad(t *testing.T) {
	cases := []string{"", "short", "not-a-valid-value-at-all!!!!"}
	for _, c := range cases {
		t.Setenv("ANARCHY_CONTACT_KEY", c)
		if _, err := LoadContactKey(); err == nil {
			t.Fatalf("key %q must be rejected", c)
		}
	}
	os.Unsetenv("ANARCHY_CONTACT_KEY")
	if _, err := LoadContactKey(); err == nil {
		t.Fatal("missing key must be rejected")
	}
}
