package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/infrastructure/auth"
)

// stubProfileRepo embeds the interface so only the methods a test exercises need
// implementing.
type stubProfileRepo struct {
	domain.ProfileRepository
	profile *domain.Profile
	hash    string
	updated string
}

func (s *stubProfileRepo) GetByIDWithHash(context.Context, uuid.UUID) (*domain.Profile, string, error) {
	return s.profile, s.hash, nil
}

func (s *stubProfileRepo) UpdatePasswordHash(_ context.Context, _ uuid.UUID, hash string) error {
	s.updated = hash
	return nil
}

// stubRefreshRepo records revocation so the test can assert other sessions die.
type stubRefreshRepo struct {
	domain.RefreshTokenRepository
	revokedFor uuid.UUID
	created    int
}

func (s *stubRefreshRepo) RevokeAllForUser(_ context.Context, userID uuid.UUID) error {
	s.revokedFor = userID
	return nil
}

func (s *stubRefreshRepo) Create(context.Context, *domain.RefreshToken) error {
	s.created++
	return nil
}

func newAuthUsecaseForTest(profile *domain.Profile, hash string) (*AuthUsecase, *stubProfileRepo, *stubRefreshRepo) {
	pr := &stubProfileRepo{profile: profile, hash: hash}
	rr := &stubRefreshRepo{}
	return NewAuthUsecase(pr, rr, []byte("test-secret")), pr, rr
}

func hashOf(t *testing.T, pw string) string {
	t.Helper()
	h, err := auth.HashPassword(pw)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return h
}

func TestChangePassword_RejectsWrongCurrent(t *testing.T) {
	u, _, _ := newAuthUsecaseForTest(&domain.Profile{ID: uuid.New(), Email: "a@b.co"}, hashOf(t, "OldPass123"))

	_, err := u.ChangePassword(context.Background(), uuid.New(), "wrong-current", "NewPass123", "ua")
	if !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("want ErrWrongPassword, got %v", err)
	}
}

func TestChangePassword_RejectsReuse(t *testing.T) {
	u, pr, _ := newAuthUsecaseForTest(&domain.Profile{ID: uuid.New(), Email: "a@b.co"}, hashOf(t, "SamePass123"))

	_, err := u.ChangePassword(context.Background(), uuid.New(), "SamePass123", "SamePass123", "ua")
	if !errors.Is(err, ErrPasswordSame) {
		t.Fatalf("want ErrPasswordSame, got %v", err)
	}
	if pr.updated != "" {
		t.Fatal("password must not be written when the new one equals the old")
	}
}

func TestChangePassword_RejectsShort(t *testing.T) {
	u, _, _ := newAuthUsecaseForTest(&domain.Profile{ID: uuid.New(), Email: "a@b.co"}, hashOf(t, "OldPass123"))

	_, err := u.ChangePassword(context.Background(), uuid.New(), "OldPass123", "short", "ua")
	if !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("want ErrWeakPassword, got %v", err)
	}
}

func TestChangePassword_SuccessStoresHashAndRevokesOthers(t *testing.T) {
	uid := uuid.New()
	u, pr, rr := newAuthUsecaseForTest(&domain.Profile{ID: uid, Email: "a@b.co"}, hashOf(t, "OldPass123"))

	pair, err := u.ChangePassword(context.Background(), uid, "OldPass123", "NewPass456", "ua")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pr.updated == "" {
		t.Fatal("password hash was not updated")
	}
	if !auth.CheckPassword(pr.updated, "NewPass456") {
		t.Fatal("stored hash does not verify the new password")
	}
	if rr.revokedFor != uid {
		t.Fatalf("expected other sessions revoked for %s, got %s", uid, rr.revokedFor)
	}
	// A fresh pair is issued so the caller's own session survives.
	if pair == nil || pair.AccessToken == "" || rr.created != 1 {
		t.Fatal("expected a fresh token pair for the current session")
	}
}
