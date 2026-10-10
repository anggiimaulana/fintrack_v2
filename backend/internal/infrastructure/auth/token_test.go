package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIssueAndParseAccessToken(t *testing.T) {
	secret := []byte("test-secret-at-least-32-characters-long")
	uid := uuid.New()

	tok, err := IssueAccessToken(uid, "user", "budi@example.com", secret, 15*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}

	claims, err := ParseAccessToken(tok, secret)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if claims.Sub != uid.String() {
		t.Errorf("sub = %q, want %q", claims.Sub, uid.String())
	}
	if claims.Role != "user" {
		t.Errorf("role = %q, want user", claims.Role)
	}
	if claims.Email != "budi@example.com" {
		t.Errorf("email = %q", claims.Email)
	}
}

func TestParseAccessTokenExpired(t *testing.T) {
	secret := []byte("test-secret-at-least-32-characters-long")
	tok, err := IssueAccessToken(uuid.New(), "user", "x@y.z", secret, -1*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	if _, err := ParseAccessToken(tok, secret); err == nil {
		t.Fatal("expected expired token to fail parsing")
	}
}

func TestParseAccessTokenWrongSecret(t *testing.T) {
	tok, _ := IssueAccessToken(uuid.New(), "user", "x@y.z", []byte("secret-one-at-least-32-characters-long"), time.Minute)
	if _, err := ParseAccessToken(tok, []byte("secret-two-at-least-32-characters-long")); err == nil {
		t.Fatal("expected token signed with a different secret to fail")
	}
}

func TestParseAccessTokenWrongAudience(t *testing.T) {
	secret := []byte("test-secret-at-least-32-characters-long")
	// A token issued for a different audience must be rejected.
	tok := mintToken(t, secret, "some-other-app")
	if _, err := ParseAccessToken(tok, secret); err == nil {
		t.Fatal("expected wrong audience to be rejected")
	}
}

func TestRefreshTokenRoundTrip(t *testing.T) {
	raw, hash := NewRefreshToken()
	if raw == "" || hash == "" {
		t.Fatal("NewRefreshToken returned empty values")
	}
	if HashRefreshToken(raw) != hash {
		t.Fatal("HashRefreshToken(raw) must equal the returned hash")
	}
	raw2, _ := NewRefreshToken()
	if raw2 == raw {
		t.Fatal("refresh tokens must be unique")
	}
}
