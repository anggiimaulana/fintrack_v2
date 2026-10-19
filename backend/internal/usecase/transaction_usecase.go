package usecase

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/internal/currency"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/infrastructure/embedding"
	"github.com/fintrack/backend/pkg/logger"
)

type TransactionUsecase struct {
	txRepo      domain.TransactionRepository
	spkQueue    *SPKQueue
	embedding   embedding.Embedder
	embedRepo   domain.EmbeddingRepository
	catNameFunc func(ctx context.Context, tx *domain.Transaction) string
	// currencySvc converts amounts to base (IDR) and resolves the user's display
	// currency. Both optional: nil keeps the IDR-only behaviour (tests).
	currencySvc      *currency.Service
	displayCurrency  func(ctx context.Context, userID uuid.UUID) string
}

func NewTransactionUsecase(
	txRepo domain.TransactionRepository,
	spkQueue *SPKQueue,
	embedder embedding.Embedder,
	embedRepo domain.EmbeddingRepository,
	catNameFunc func(ctx context.Context, tx *domain.Transaction) string,
) *TransactionUsecase {
	return &TransactionUsecase{
		txRepo:      txRepo,
		spkQueue:    spkQueue,
		embedding:   embedder,
		embedRepo:   embedRepo,
		catNameFunc: catNameFunc,
	}
}

// WithCurrency attaches the conversion service and a resolver for the user's
// display currency. Returns the usecase for chaining.
func (u *TransactionUsecase) WithCurrency(svc *currency.Service, display func(ctx context.Context, userID uuid.UUID) string) *TransactionUsecase {
	u.currencySvc = svc
	u.displayCurrency = display
	return u
}

// resolveCurrency returns the effective currency for a write: the explicit code
// when given, else the user's display currency, else IDR.
func (u *TransactionUsecase) resolveCurrency(ctx context.Context, userID uuid.UUID, explicit string) string {
	if code := strings.ToUpper(strings.TrimSpace(explicit)); code != "" {
		return code
	}
	if u.displayCurrency != nil {
		if c := u.displayCurrency(ctx, userID); c != "" {
			return c
		}
	}
	return currency.BaseCode
}

// applyRate snapshots the rate for a currency at a moment. For the base currency
// it leaves rate nil. No-op (IDR semantics) when the service is absent.
func (u *TransactionUsecase) applyRate(ctx context.Context, tx *domain.Transaction, code string, at time.Time) error {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		code = currency.BaseCode
	}
	tx.CurrencyCode = code
	if code == currency.BaseCode || u.currencySvc == nil {
		tx.ExchangeRate = 1
		tx.RateAt = nil
		return nil
	}
	// RateForWrite validates the currency is active and tolerates dates that
	// predate the rate history (backdated entries).
	rate, err := u.currencySvc.RateForWrite(ctx, code, at)
	if err != nil {
		return err
	}
	tx.ExchangeRate = rate.RateToBase
	rt := rate.EffectiveAt
	tx.RateAt = &rt
	return nil
}
// baseOverflowLimit is just under the largest value the numeric(20,2)
// amount_base column can store.
const baseOverflowLimit = 9e17

// guardBaseOverflow rejects a write whose computed base value would overflow the
// generated amount_base column. Without it, Postgres raises a numeric-overflow
// error and the user sees a 500 for a value they could have fixed.
func guardBaseOverflow(amount, rate float64) error {
	base := amount * rate
	if math.IsNaN(base) || math.IsInf(base, 0) || base > baseOverflowLimit {
		return fmt.Errorf("%w: nominal terlalu besar untuk dikonversi", domain.ErrInvalidAmount)
	}
	return nil
}

func (u *TransactionUsecase) Create(ctx context.Context, userID uuid.UUID, req domain.CreateTransactionRequest) (*domain.Transaction, error) {
	tx := &domain.Transaction{
		UserID:            userID,
		Type:              req.Type,
		Amount:            req.Amount,
		IncomeCategoryID:  req.IncomeCategoryID,
		ExpenseCategoryID: req.ExpenseCategoryID,
		SavingsCategoryID: req.SavingsCategoryID,
		SubcategoryID:     req.SubcategoryID,
		Description:       req.Description,
		Notes:             req.Notes,
		Tags:              req.Tags,
		Source:            req.Source,
	}

	if req.TransactionDate != nil {
		tx.TransactionDate = *req.TransactionDate
	} else {
		tx.TransactionDate = time.Now()
	}

	if req.IsRecurring {
		tx.IsRecurring = true
		tx.RecurringInterval = req.RecurringInterval
	}

	if tx.Source == "" {
		tx.Source = domain.SourceManual
	}

	code := u.resolveCurrency(ctx, userID, req.CurrencyCode)
	if err := u.applyRate(ctx, tx, code, tx.TransactionDate); err != nil {
		return nil, err
	}
	if err := guardBaseOverflow(tx.Amount, tx.ExchangeRate); err != nil {
		return nil, err
	}

	if err := u.txRepo.Create(ctx, tx); err != nil {
		return nil, err
	}

	if u.spkQueue != nil {
		u.spkQueue.Enqueue(userID)
	}
	u.embedAndStore(ctx, tx)

	return tx, nil
}

func (u *TransactionUsecase) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Transaction, error) {
	return u.txRepo.GetByID(ctx, id, userID)
}

func (u *TransactionUsecase) List(ctx context.Context, userID uuid.UUID, filter domain.TransactionFilter) ([]domain.Transaction, int, error) {
	return u.txRepo.List(ctx, userID, filter)
}

// Update applies changes with optimistic locking when req.UpdatedAt is set.
func (u *TransactionUsecase) Update(ctx context.Context, userID uuid.UUID, id uuid.UUID, req domain.UpdateTransactionRequest) (*domain.Transaction, error) {
	tx, err := u.txRepo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if tx == nil {
		return nil, nil
	}

	if req.Amount != nil {
		// Guard the write path: Create enforces gt=0 via binding, Update does not.
		if *req.Amount <= 0 || math.IsNaN(*req.Amount) || math.IsInf(*req.Amount, 0) {
			return nil, fmt.Errorf("%w: nominal harus berupa angka positif", domain.ErrInvalidAmount)
		}
		tx.Amount = *req.Amount
	}
	if req.IncomeCategoryID != nil {
		tx.IncomeCategoryID = req.IncomeCategoryID
	}
	if req.ExpenseCategoryID != nil {
		tx.ExpenseCategoryID = req.ExpenseCategoryID
	}
	if req.SavingsCategoryID != nil {
		tx.SavingsCategoryID = req.SavingsCategoryID
	}
	if req.SubcategoryID != nil {
		tx.SubcategoryID = req.SubcategoryID
	}
	if req.Description != nil {
		tx.Description = req.Description
	}
	if req.Notes != nil {
		tx.Notes = req.Notes
	}
	if req.Tags != nil {
		tx.Tags = req.Tags
	}
	if req.TransactionDate != nil {
		tx.TransactionDate = *req.TransactionDate
	}
	if req.IsRecurring != nil {
		tx.IsRecurring = *req.IsRecurring
	}
	if req.RecurringInterval != nil {
		tx.RecurringInterval = req.RecurringInterval
	}
	if req.Source != nil {
		tx.Source = *req.Source
	}

	// Currency semantics: changing only the amount keeps the transaction's
	// historical rate; changing the currency re-quotes at the transaction date so
	// the base value reflects the currency actually recorded.
	if tx.CurrencyCode == "" {
		tx.CurrencyCode = currency.BaseCode
	}
	amountChanged := req.Amount != nil
	newCode := tx.CurrencyCode
	currencyChanged := false
	if req.CurrencyCode != nil {
		if c := strings.ToUpper(strings.TrimSpace(*req.CurrencyCode)); c != "" && c != tx.CurrencyCode {
			newCode = c
			currencyChanged = true
		}
	}
	if currencyChanged {
		if err := u.applyRate(ctx, tx, newCode, tx.TransactionDate); err != nil {
			return nil, err
		}
	} else if amountChanged && tx.CurrencyCode == currency.BaseCode {
		// Base currency: rate stays 1 (generated amount_base follows amount).
		tx.ExchangeRate = 1
		tx.RateAt = nil
	}

	if err := guardBaseOverflow(tx.Amount, tx.ExchangeRate); err != nil {
		return nil, err
	}

	if err := u.txRepo.Update(ctx, tx, req.UpdatedAt); err != nil {
		return nil, err
	}

	if u.spkQueue != nil {
		u.spkQueue.Enqueue(userID)
	}
	u.embedAndStore(ctx, tx)

	return tx, nil
}

func (u *TransactionUsecase) SoftDelete(ctx context.Context, id, userID uuid.UUID) error {
	if err := u.txRepo.SoftDelete(ctx, id, userID); err != nil {
		return err
	}

	if u.spkQueue != nil {
		u.spkQueue.Enqueue(userID)
	}
	if u.embedRepo != nil {
		if err := u.embedRepo.DeleteByTransactionID(ctx, id); err != nil {
			logger.Warn("failed to delete embedding", zap.Error(err), zap.String("transaction_id", id.String()))
		}
	}

	return nil
}

func (u *TransactionUsecase) GetStats(ctx context.Context, userID uuid.UUID, period domain.StatsPeriod) (*domain.TransactionStats, error) {
	return u.txRepo.GetStats(ctx, userID, period)
}

// buildEmbeddingText produces the canonical text embedded for a transaction:
// "{transaction_date} {type} {category} {description} {CURRENCY}{amount}".
// The currency code is included so semantic search keeps the unit.
func buildEmbeddingText(tx *domain.Transaction, category string) string {
	desc := ""
	if tx.Description != nil {
		desc = *tx.Description
	}
	code := tx.CurrencyCode
	if code == "" {
		code = currency.BaseCode
	}
	return fmt.Sprintf("%s %s %s %s %s%.0f",
		tx.TransactionDate.Format("2006-01-02"), tx.Type, category, desc, code, tx.Amount)
}

// embedAndStore is best-effort: a failure to embed never fails the mutation.
func (u *TransactionUsecase) embedAndStore(ctx context.Context, tx *domain.Transaction) {
	if u.embedding == nil || u.embedRepo == nil || !u.embedding.Enabled() {
		return
	}

	category := ""
	if u.catNameFunc != nil {
		category = u.catNameFunc(ctx, tx)
	}

	content := buildEmbeddingText(tx, category)

	vec, err := u.embedding.Embed(ctx, content)
	if err != nil {
		logger.Warn("embedding failed; transaction kept", zap.Error(err), zap.String("transaction_id", tx.ID.String()))
		return
	}

	metadata := map[string]interface{}{
		"transaction_id": tx.ID.String(),
		"date":           tx.TransactionDate.Format("2006-01-02"),
		"type":           string(tx.Type),
		"category":       category,
		"amount":         tx.Amount,
		"currency_code":  tx.CurrencyCode,
	}

	if err := u.embedRepo.Upsert(ctx, tx.UserID, tx.ID, content, vec, metadata); err != nil {
		logger.Warn("failed to store embedding", zap.Error(err), zap.String("transaction_id", tx.ID.String()))
	}
}
