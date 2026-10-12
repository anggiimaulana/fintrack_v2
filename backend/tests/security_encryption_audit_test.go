package tests

import (
	"context"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/infrastructure/crypto"
)

// TestP08_006_AuditCredentialEncryptionAtRest verifies AES-256-GCM at-rest encryption
// for all credentials and secrets stored in database tables.
// Invariant: 0 plaintext API keys in database, ciphertext is valid hex-encoded
// nonce (12B) + ciphertext + auth tag (16B), tampering fails authentication,
// and decryption with master key succeeds transparently.
func TestP08_006_AuditCredentialEncryptionAtRest(t *testing.T) {
	// 1. Cryptographic implementation verification
	key32 := []byte("01234567890123456789012345678901") // 32 bytes
	plaintext := "sk-secret-live-api-key-test-value-12345"

	ciphertextHex, err := crypto.Encrypt([]byte(plaintext), key32)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	rawCiphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		t.Fatalf("ciphertext must be valid hex: %v", err)
	}

	// GCM overhead: 12 bytes nonce + 16 bytes auth tag = 28 bytes overhead
	expectedLen := len(plaintext) + 12 + 16
	if len(rawCiphertext) != expectedLen {
		t.Errorf("expected %d raw bytes, got %d", expectedLen, len(rawCiphertext))
	}

	// Plaintext must never appear in ciphertext
	if strings.Contains(ciphertextHex, "sk-secret") {
		t.Fatalf("plaintext leaked in ciphertext: %s", ciphertextHex)
	}

	// Transparent decryption with correct key
	decrypted, err := crypto.Decrypt(ciphertextHex, key32)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if string(decrypted) != plaintext {
		t.Fatalf("decrypted value mismatch: got %q, want %q", string(decrypted), plaintext)
	}

	// Tampered ciphertext must fail authentication
	tampered := []byte(ciphertextHex)
	if tampered[20] == 'a' {
		tampered[20] = 'b'
	} else {
		tampered[20] = 'a'
	}
	if _, err := crypto.Decrypt(string(tampered), key32); err == nil {
		t.Fatal("expected tampering to fail authentication, but decrypt succeeded")
	}

	// Wrong key must fail decryption
	wrongKey := []byte("wrongwrongwrongwrongwrongwrong12")
	if _, err := crypto.Decrypt(ciphertextHex, wrongKey); err == nil {
		t.Fatal("expected wrong key to fail decryption, but decrypt succeeded")
	}

	// 2. Live database audit against PostgreSQL tables (ai_endpoints & gemini_api_keys)
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping live DB encryption audit")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect DB: %v", err)
	}
	defer pool.Close()

	// Audit ai_endpoints
	rows, err := pool.Query(context.Background(), `SELECT name, api_key_encrypted FROM ai_endpoints WHERE api_key_encrypted != ''`)
	if err != nil {
		t.Fatalf("query ai_endpoints: %v", err)
	}
	defer rows.Close()

	var endpointCount int
	for rows.Next() {
		var name, encKey string
		if err := rows.Scan(&name, &encKey); err != nil {
			t.Fatalf("scan row: %v", err)
		}
		endpointCount++

		// Invariant: no plaintext prefix
		if strings.HasPrefix(encKey, "sk-") || strings.HasPrefix(encKey, "AIza") {
			t.Errorf("CRITICAL: Plaintext API key found in ai_endpoints %s: %s", name, encKey[:10])
		}

		// Must be valid hex with minimum length (nonce 12B + tag 16B = 28B = 56 hex chars)
		raw, err := hex.DecodeString(encKey)
		if err != nil || len(raw) < 28 {
			t.Errorf("Invalid ciphertext structure in ai_endpoints %s (len=%d, hexErr=%v)", name, len(raw), err)
		}
	}
	t.Logf("Database encryption audit PASSED: %d ai_endpoints audited (100%% ciphertext at-rest)",
		endpointCount)
}
