package auth

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("User@123456")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "User@123456" {
		t.Fatal("hash must not equal plaintext")
	}
	if !CheckPassword(hash, "User@123456") {
		t.Fatal("expected password to verify")
	}
	if CheckPassword(hash, "wrong-password") {
		t.Fatal("expected wrong password to fail")
	}
}

func TestHashPasswordIsSalted(t *testing.T) {
	h1, _ := HashPassword("same-password")
	h2, _ := HashPassword("same-password")
	if h1 == h2 {
		t.Fatal("bcrypt hashes of the same password should differ (random salt)")
	}
}
