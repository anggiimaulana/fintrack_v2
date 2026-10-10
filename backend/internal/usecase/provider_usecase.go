package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
)

type ProviderUsecase struct {
	providerRepo domain.AIProviderRepository
}

func NewProviderUsecase(providerRepo domain.AIProviderRepository) *ProviderUsecase {
	return &ProviderUsecase{providerRepo: providerRepo}
}

func (u *ProviderUsecase) List(ctx context.Context) ([]domain.AIProvider, error) {
	providers, err := u.providerRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	if providers == nil {
		providers = []domain.AIProvider{}
	}
	return providers, nil
}

func (u *ProviderUsecase) GetByName(ctx context.Context, name string) (*domain.AIProvider, error) {
	return u.providerRepo.GetByName(ctx, name)
}

func (u *ProviderUsecase) Create(ctx context.Context, adminID uuid.UUID, req domain.CreateProviderRequest) (*domain.AIProvider, error) {
	existing, err := u.providerRepo.GetByName(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("provider with name '%s' already exists", req.Name)
	}

	provider := &domain.AIProvider{
		Name:            req.Name,
		DisplayName:     req.DisplayName,
		SupportsFast:    req.SupportsFast,
		SupportsReasoning: req.SupportsReasoning,
		IsActive:        true,
	}
	if !req.IsActive {
		provider.IsActive = false
	}

	if err := u.providerRepo.Create(ctx, provider); err != nil {
		return nil, err
	}
	return provider, nil
}

func (u *ProviderUsecase) Update(ctx context.Context, id uuid.UUID, req domain.UpdateProviderRequest) (*domain.AIProvider, error) {
	provider, err := u.providerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, fmt.Errorf("provider not found")
	}

	if req.Name != nil {
		provider.Name = *req.Name
	}
	if req.DisplayName != nil {
		provider.DisplayName = *req.DisplayName
	}
	if req.SupportsFast != nil {
		provider.SupportsFast = *req.SupportsFast
	}
	if req.SupportsReasoning != nil {
		provider.SupportsReasoning = *req.SupportsReasoning
	}
	if req.IsActive != nil {
		provider.IsActive = *req.IsActive
	}

	if err := u.providerRepo.Update(ctx, provider); err != nil {
		return nil, err
	}
	return provider, nil
}
