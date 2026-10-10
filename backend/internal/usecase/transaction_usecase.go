package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/infrastructure/embedding"
	"github.com/fintrack/backend/pkg/logger"
)

type TransactionUsecase struct {
	txRepo      domain.TransactionRepository
	spkQueue    *SPKQueue
	embedding   *embedding.GeminiEmbedder
	embedRepo   domain.EmbeddingRepository
	catNameFunc func(ctx context.Context, tx *domain.Transaction) string
}

func NewTransactionUsecase(
	txRepo domain.TransactionRepository,
	spkQueue *SPKQueue,
	embedder *embedding.GeminiEmbedder,
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

	if err := u.txRepo.Create(ctx, tx); err != nil {
		return nil, err
	}

	u.spkQueue.Enqueue(userID)
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

	if err := u.txRepo.Update(ctx, tx, req.UpdatedAt); err != nil {
		return nil, err
	}

	u.spkQueue.Enqueue(userID)
	u.embedAndStore(ctx, tx)

	return tx, nil
}

func (u *TransactionUsecase) SoftDelete(ctx context.Context, id, userID uuid.UUID) error {
	if err := u.txRepo.SoftDelete(ctx, id, userID); err != nil {
		return err
	}

	u.spkQueue.Enqueue(userID)
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

// embedAndStore is best-effort: a failure to embed never fails the mutation.
func (u *TransactionUsecase) embedAndStore(ctx context.Context, tx *domain.Transaction) {
	if u.embedding == nil || u.embedRepo == nil || !u.embedding.Enabled() {
		return
	}

	category := ""
	if u.catNameFunc != nil {
		category = u.catNameFunc(ctx, tx)
	}
	desc := ""
	if tx.Description != nil {
		desc = *tx.Description
	}

	content := fmt.Sprintf("%s %s %s %s Rp%.0f",
		tx.TransactionDate.Format("2006-01-02"), tx.Type, category, desc, tx.Amount)

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
	}

	if err := u.embedRepo.Upsert(ctx, tx.UserID, tx.ID, content, vec, metadata); err != nil {
		logger.Warn("failed to store embedding", zap.Error(err), zap.String("transaction_id", tx.ID.String()))
	}
}
