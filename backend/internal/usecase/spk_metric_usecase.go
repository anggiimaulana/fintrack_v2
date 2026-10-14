package usecase

import (
	"context"

	"github.com/fintrack/backend/internal/domain"
)

// SPKMetricUsecase owns the SPK metric *configuration* lifecycle: the
// super-admin default model and per-user overrides. Scoring itself lives in
// SPKUsecase; this type only validates and persists configuration.
type SPKMetricUsecase struct {
	repo domain.SPKMetricRepository
}

func NewSPKMetricUsecase(repo domain.SPKMetricRepository) *SPKMetricUsecase {
	return &SPKMetricUsecase{repo: repo}
}

// GetDefault returns the active super-admin default, falling back to the
// compiled model when none is seeded so the admin UI always has something to
// show and edit.
func (u *SPKMetricUsecase) GetDefault(ctx context.Context) (domain.SPKMetricModel, error) {
	def, err := u.repo.GetActiveDefault(ctx)
	if err != nil {
		return domain.SPKMetricModel{}, err
	}
	if def == nil || len(def.Indicators) == 0 {
		return domain.DefaultSPKMetricModel(), nil
	}
	def.Source = "default"
	return *def, nil
}

// SetDefault validates and replaces the active default model. Replacing (rather
// than mutating in place) keeps a history of prior models for rollback.
func (u *SPKMetricUsecase) SetDefault(ctx context.Context, name, description string, m domain.SPKMetricModel, updatedBy *string) (domain.SPKMetricModel, error) {
	if err := domain.ValidateMetricModel(m); err != nil {
		return domain.SPKMetricModel{}, err
	}
	if name == "" {
		name = "Model SPK Default"
	}
	out, err := u.repo.ReplaceDefault(ctx, name, description, m, updatedBy)
	if err != nil {
		return domain.SPKMetricModel{}, err
	}
	return *out, nil
}

// GetUserModel returns the model that applies to the user plus a flag saying
// whether it is a personal override. The UI uses `isOverride` to show the
// "using default" vs "customised" state.
func (u *SPKMetricUsecase) GetUserModel(ctx context.Context, userID string) (domain.SPKMetricModel, bool, error) {
	override, err := u.repo.GetUserOverride(ctx, userID)
	if err != nil {
		return domain.SPKMetricModel{}, false, err
	}
	if override != nil && len(override.Indicators) > 0 {
		override.Source = "user"
		return *override, true, nil
	}
	def, err := u.GetDefault(ctx)
	if err != nil {
		return domain.SPKMetricModel{}, false, err
	}
	return def, false, nil
}

// SetUserOverride validates and stores a user's personal model.
func (u *SPKMetricUsecase) SetUserOverride(ctx context.Context, userID string, m domain.SPKMetricModel) (domain.SPKMetricModel, error) {
	if err := domain.ValidateMetricModel(m); err != nil {
		return domain.SPKMetricModel{}, err
	}
	if err := u.repo.UpsertUserOverride(ctx, userID, m); err != nil {
		return domain.SPKMetricModel{}, err
	}
	m.Source = "user"
	return m, nil
}

// DeleteUserOverride reverts the user to the super-admin default.
func (u *SPKMetricUsecase) DeleteUserOverride(ctx context.Context, userID string) error {
	return u.repo.DeleteUserOverride(ctx, userID)
}
