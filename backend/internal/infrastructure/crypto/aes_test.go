package crypto

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{"short text", "hello"},
		{"api key format", "AIzaSyAbCdEfGhIjKlMnOpQrStUvWxYz123456789"},
		{"empty string", ""},
		{"with special chars", "test_key_123!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"long text", "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := Encrypt([]byte(tt.plaintext), key)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}
			if ciphertext == "" {
				t.Fatal("Encrypt() returned empty string")
			}

			decrypted, err := Decrypt(ciphertext, key)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}
			if string(decrypted) != tt.plaintext {
				t.Fatalf("Decrypt() = %q, want %q", string(decrypted), tt.plaintext)
			}
		})
	}
}

func TestEncryptInvalidKey(t *testing.T) {
	_, err := Encrypt([]byte("test"), []byte("short"))
	if err != ErrInvalidKeyLength {
		t.Fatalf("Expected ErrInvalidKeyLength, got %v", err)
	}
}

func TestDecryptInvalidKey(t *testing.T) {
	_, err := Decrypt("abc123", []byte("short"))
	if err != ErrInvalidKeyLength {
		t.Fatalf("Expected ErrInvalidKeyLength, got %v", err)
	}
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	key := make([]byte, 32)
	_, err := Decrypt("invalidhex", key)
	if err == nil {
		t.Fatal("Expected error for invalid ciphertext, got nil")
	}
}

func TestEncryptUniqueOutput(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := []byte("same text every time")
	c1, _ := Encrypt(plaintext, key)
	c2, _ := Encrypt(plaintext, key)

	if c1 == c2 {
		t.Fatal("Encrypt should produce unique output each time (nonce)")
	}
}
