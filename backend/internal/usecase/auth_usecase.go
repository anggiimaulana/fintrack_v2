package usecase

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/infrastructure/auth"
)

// Auth error sentinels mapped to HTTP statuses by the handler.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountInactive    = errors.New("account inactive")
	ErrPasswordReset      = errors.New("password reset required")
	ErrInvalidRefresh     = errors.New("invalid refresh token")
	ErrWeakPassword       = errors.New("password too short")
	ErrInvalidEmail       = errors.New("invalid email")
	ErrSelfDeactivate     = errors.New("cannot deactivate own account")
)

var emailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// TokenPair is returned by login/refresh.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	User         *domain.Profile
}

const (
	accessTTL  = 1 * time.Hour
	refreshTTL = 30 * 24 * time.Hour
)

// AuthUsecase implements Go-native authentication: bcrypt passwords, HS256
// access tokens, and rotating refresh tokens persisted in the database.
type AuthUsecase struct {
	userRepo    domain.ProfileRepository
	refreshRepo domain.RefreshTokenRepository
	jwtSecret   []byte
}

func NewAuthUsecase(userRepo domain.ProfileRepository, refreshRepo domain.RefreshTokenRepository, jwtSecret []byte) *AuthUsecase {
	return &AuthUsecase{userRepo: userRepo, refreshRepo: refreshRepo, jwtSecret: jwtSecret}
}

func (u *AuthUsecase) issuePair(ctx context.Context, p *domain.Profile, userAgent string) (*TokenPair, error) {
	access, err := auth.IssueAccessToken(p.ID, string(p.Role), p.Email, u.jwtSecret, accessTTL)
	if err != nil {
		return nil, err
	}
	raw, hash := auth.NewRefreshToken()
	rt := &domain.RefreshToken{
		UserID:    p.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(refreshTTL),
	}
	if userAgent != "" {
		rt.UserAgent = &userAgent
	}
	if err := u.refreshRepo.Create(ctx, rt); err != nil {
		return nil, err
	}
	return &TokenPair{AccessToken: access, RefreshToken: raw, ExpiresIn: int(accessTTL.Seconds()), User: p}, nil
}

// Register creates a new user (role 'user') and returns the profile.
func (u *AuthUsecase) Register(ctx context.Context, email, password, fullName string) (*domain.Profile, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !emailRegex.MatchString(email) {
		return nil, ErrInvalidEmail
	}
	if len(password) < 8 {
		return nil, ErrWeakPassword
	}

	existing, _, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrEmailTaken
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	if fullName == "" {
		fullName = strings.Split(email, "@")[0]
	}

	return u.userRepo.Create(ctx, &domain.Profile{
		Email:    email,
		FullName: fullName,
		Role:     domain.RoleUser,
	}, hash)
}

// Login verifies credentials and issues a token pair.
func (u *AuthUsecase) Login(ctx context.Context, email, password, userAgent string) (*TokenPair, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	p, hash, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrInvalidCredentials
	}
	if !p.IsActive {
		return nil, ErrAccountInactive
	}
	if hash == "" {
		return nil, ErrPasswordReset
	}
	if !auth.CheckPassword(hash, password) {
		return nil, ErrInvalidCredentials
	}
	return u.issuePair(ctx, p, userAgent)
}

// ChangePassword verifies the current password, rejects reusing the same one,
// enforces the same minimum length as registration, stores a fresh bcrypt hash,
// and revokes every other refresh token so a stolen session cannot outlive the
// change. The caller's own session is re-issued so they stay signed in.
var (
	ErrWrongPassword  = errors.New("current password is incorrect")
	ErrPasswordSame   = errors.New("new password must differ from the current one")
)

func (u *AuthUsecase) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword, userAgent string) (*TokenPair, error) {
	if len(newPassword) < 8 {
		return nil, ErrWeakPassword
	}

	p, hash, err := u.userRepo.GetByIDWithHash(ctx, userID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrInvalidCredentials
	}
	if hash == "" || !auth.CheckPassword(hash, currentPassword) {
		return nil, ErrWrongPassword
	}
	// Reject reuse: a no-op "change" would leave a compromised password valid.
	if auth.CheckPassword(hash, newPassword) {
		return nil, ErrPasswordSame
	}

	newHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return nil, err
	}
	if err := u.userRepo.UpdatePasswordHash(ctx, userID, newHash); err != nil {
		return nil, err
	}
	// Revoke every existing refresh token, then issue a fresh pair for this
	// device so other sessions are logged out but the current one survives.
	if err := u.refreshRepo.RevokeAllForUser(ctx, userID); err != nil {
		return nil, err
	}
	return u.issuePair(ctx, p, userAgent)
}

// Refresh rotates a refresh token. If a revoked token is presented, it is
// treated as reuse: all of the user's tokens are revoked and an error returned.
func (u *AuthUsecase) Refresh(ctx context.Context, rawToken, userAgent string) (*TokenPair, error) {
	if rawToken == "" {
		return nil, ErrInvalidRefresh
	}
	hash := auth.HashRefreshToken(rawToken)
	existing, err := u.refreshRepo.GetByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if existing == nil || time.Now().After(existing.ExpiresAt) {
		return nil, ErrInvalidRefresh
	}
	if existing.RevokedAt != nil {
		// Reuse detected: revoke everything for this user.
		_ = u.refreshRepo.RevokeAllForUser(ctx, existing.UserID)
		return nil, ErrInvalidRefresh
	}

	p, err := u.userRepo.GetByID(ctx, existing.UserID)
	if err != nil {
		return nil, err
	}
	if p == nil || !p.IsActive {
		return nil, ErrInvalidRefresh
	}

	access, err := auth.IssueAccessToken(p.ID, string(p.Role), p.Email, u.jwtSecret, accessTTL)
	if err != nil {
		return nil, err
	}
	raw, newHash := auth.NewRefreshToken()
	newToken := &domain.RefreshToken{
		UserID:    p.ID,
		TokenHash: newHash,
		ExpiresAt: time.Now().Add(refreshTTL),
	}
	if userAgent != "" {
		newToken.UserAgent = &userAgent
	}
	if err := u.refreshRepo.Rotate(ctx, existing.ID, newToken); err != nil {
		if errors.Is(err, domain.ErrRefreshTokenRevoked) {
			_ = u.refreshRepo.RevokeAllForUser(ctx, existing.UserID)
			return nil, ErrInvalidRefresh
		}
		return nil, err
	}
	return &TokenPair{AccessToken: access, RefreshToken: raw, ExpiresIn: int(accessTTL.Seconds()), User: p}, nil
}

// Logout revokes the presented refresh token.
func (u *AuthUsecase) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	existing, err := u.refreshRepo.GetByHash(ctx, auth.HashRefreshToken(rawToken))
	if err != nil {
		return err
	}
	if existing == nil || existing.RevokedAt != nil {
		return nil
	}
	// Rotate-free revocation: create no successor.
	return u.refreshRepo.RevokeAllForUser(ctx, existing.UserID)
}

// Me returns the current user's profile.
func (u *AuthUsecase) Me(ctx context.Context, userID uuid.UUID) (*domain.Profile, error) {
	return u.userRepo.GetByID(ctx, userID)
}

// UpdateProfile replaces the current user's editable profile fields. It is a
// full update (PUT semantics): an empty phone/avatar clears the field, which is
// how the UI removes a value.
func (u *AuthUsecase) UpdateProfile(ctx context.Context, userID uuid.UUID, fullName string, phone *string, avatarURL *string) (*domain.Profile, error) {
	p, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}
	p.FullName = fullName
	p.Phone = phone
	p.AvatarURL = avatarURL
	if err := u.userRepo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// SetAvatarURL updates only the avatar, leaving other profile fields untouched.
func (u *AuthUsecase) SetAvatarURL(ctx context.Context, userID uuid.UUID, url string) (*domain.Profile, error) {
	p, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}
	p.AvatarURL = &url
	if err := u.userRepo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}
