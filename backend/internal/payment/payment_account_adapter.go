package payment

import (
	"context"

	"github.com/fintrack/backend/internal/domain"
)

// PaymentAccountAdapter bridges the domain payment-account repository to the
// payment package's narrow PaymentAccountSource interface, mapping the domain
// type into the package-local ManualAccount shape. Keeping the adapter here (in
// payment) means the payment package never imports domain, avoiding a cycle.
type PaymentAccountAdapter struct {
	repo domain.PaymentAccountRepository
}

// NewPaymentAccountAdapter wraps a repository as a PaymentAccountSource.
func NewPaymentAccountAdapter(repo domain.PaymentAccountRepository) *PaymentAccountAdapter {
	return &PaymentAccountAdapter{repo: repo}
}

// ListActive returns the active accounts in the shape the payment package uses.
func (a *PaymentAccountAdapter) ListActive(ctx context.Context) ([]ManualAccount, error) {
	accounts, err := a.repo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ManualAccount, 0, len(accounts))
	for _, acc := range accounts {
		out = append(out, ManualAccount{
			Kind:          string(acc.Kind),
			Name:          acc.Name,
			AccountNumber: acc.AccountNumber,
			AccountHolder: acc.AccountHolder,
			QRISURL:       acc.QRISURL,
		})
	}
	return out, nil
}

var _ PaymentAccountSource = (*PaymentAccountAdapter)(nil)
