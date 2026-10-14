package usecase

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
)

// PaymentAccountUsecase owns manual payment destinations: the super-admin CRUD
// and the active list shown to users at checkout.
type PaymentAccountUsecase struct {
	repo domain.PaymentAccountRepository
}

func NewPaymentAccountUsecase(repo domain.PaymentAccountRepository) *PaymentAccountUsecase {
	return &PaymentAccountUsecase{repo: repo}
}

// ListActive returns the accounts users may pay to (display order).
func (u *PaymentAccountUsecase) ListActive(ctx context.Context) ([]domain.PaymentAccount, error) {
	return u.repo.ListActive(ctx)
}

// ListAll returns every account for the admin UI.
func (u *PaymentAccountUsecase) ListAll(ctx context.Context) ([]domain.PaymentAccount, error) {
	return u.repo.ListAll(ctx)
}

// Create validates and stores a new account.
func (u *PaymentAccountUsecase) Create(ctx context.Context, req domain.UpsertPaymentAccountRequest) (*domain.PaymentAccount, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.AccountNumber = strings.TrimSpace(req.AccountNumber)
	req.AccountHolder = strings.TrimSpace(req.AccountHolder)
	req.QRISURL = strings.TrimSpace(req.QRISURL)
	if req.Kind == "" {
		req.Kind = domain.PaymentAccountBank
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return u.repo.Create(ctx, req)
}

// Update validates and replaces an account.
func (u *PaymentAccountUsecase) Update(ctx context.Context, id uuid.UUID, req domain.UpsertPaymentAccountRequest) (*domain.PaymentAccount, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.AccountNumber = strings.TrimSpace(req.AccountNumber)
	req.AccountHolder = strings.TrimSpace(req.AccountHolder)
	req.QRISURL = strings.TrimSpace(req.QRISURL)
	if req.Kind == "" {
		req.Kind = domain.PaymentAccountBank
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return u.repo.Update(ctx, id, req)
}

// Delete removes an account.
func (u *PaymentAccountUsecase) Delete(ctx context.Context, id uuid.UUID) error {
	return u.repo.Delete(ctx, id)
}
