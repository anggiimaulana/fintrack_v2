package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
)

type UserUsecase struct {
	userRepo domain.ProfileRepository
}

func NewUserUsecase(userRepo domain.ProfileRepository) *UserUsecase {
	return &UserUsecase{userRepo: userRepo}
}

func (u *UserUsecase) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.Profile, error) {
	return u.userRepo.GetByID(ctx, userID)
}

func (u *UserUsecase) UpdateProfile(ctx context.Context, userID uuid.UUID, fullName *string, phone *string, avatarURL *string) (*domain.Profile, error) {
	p, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}

	if fullName != nil {
		p.FullName = *fullName
	}
	if phone != nil {
		p.Phone = phone
	}
	if avatarURL != nil {
		p.AvatarURL = avatarURL
	}

	if err := u.userRepo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (u *UserUsecase) ListUsers(ctx context.Context) ([]domain.UserStats, *domain.AdminStats, error) {
	return u.userRepo.ListWithStats(ctx)
}

// SetUserActive toggles a user's active flag, rejecting self-deactivation.
func (u *UserUsecase) SetUserActive(ctx context.Context, id uuid.UUID, isActive bool, actorID uuid.UUID) error {
	if id == actorID && !isActive {
		return ErrSelfDeactivate
	}
	return u.userRepo.SetActive(ctx, id, isActive)
}
