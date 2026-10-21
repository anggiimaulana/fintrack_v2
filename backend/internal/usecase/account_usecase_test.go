package usecase_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/usecase"
)

type memoryAccountRepo struct {
	accounts map[uuid.UUID]domain.FinancialAccount
}

func newMemoryAccountRepo() *memoryAccountRepo {
	return &memoryAccountRepo{accounts: make(map[uuid.UUID]domain.FinancialAccount)}
}

func (m *memoryAccountRepo) Create(_ context.Context, a *domain.FinancialAccount) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()
	m.accounts[a.ID] = *a
	return nil
}

func (m *memoryAccountRepo) GetByID(_ context.Context, id, _ uuid.UUID) (*domain.FinancialAccount, error) {
	a, ok := m.accounts[id]
	if !ok {
		return nil, domain.ErrFinancialAccountNotFound
	}
	return &a, nil
}

func (m *memoryAccountRepo) GetByIDForUpdate(ctx context.Context, id, userID uuid.UUID) (*domain.FinancialAccount, error) {
	return m.GetByID(ctx, id, userID)
}

func (m *memoryAccountRepo) ListByUser(_ context.Context, userID uuid.UUID, onlyActive bool) ([]domain.FinancialAccount, error) {
	var out []domain.FinancialAccount
	for _, a := range m.accounts {
		if a.UserID == userID && (!onlyActive || a.IsActive) {
			out = append(out, a)
		}
	}
	return out, nil
}

func (m *memoryAccountRepo) Update(_ context.Context, a *domain.FinancialAccount) error {
	if _, ok := m.accounts[a.ID]; !ok {
		return domain.ErrFinancialAccountNotFound
	}
	a.UpdatedAt = time.Now()
	m.accounts[a.ID] = *a
	return nil
}

func (m *memoryAccountRepo) UpdateBalance(_ context.Context, id, _ uuid.UUID, newBalance float64) error {
	a, ok := m.accounts[id]
	if !ok {
		return domain.ErrFinancialAccountNotFound
	}
	a.CurrentBalance = newBalance
	a.UpdatedAt = time.Now()
	m.accounts[id] = a
	return nil
}

func (m *memoryAccountRepo) Delete(_ context.Context, id, _ uuid.UUID) error {
	if _, ok := m.accounts[id]; !ok {
		return domain.ErrFinancialAccountNotFound
	}
	delete(m.accounts, id)
	return nil
}

func (m *memoryAccountRepo) CountByUser(_ context.Context, userID uuid.UUID) (int, error) {
	count := 0
	for _, a := range m.accounts {
		if a.UserID == userID && a.IsActive {
			count++
		}
	}
	return count, nil
}

type memoryBatchRepo struct {
	batches map[uuid.UUID]domain.ImportBatch
}

func newMemoryBatchRepo() *memoryBatchRepo {
	return &memoryBatchRepo{batches: make(map[uuid.UUID]domain.ImportBatch)}
}

func (m *memoryBatchRepo) Create(_ context.Context, b *domain.ImportBatch) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	b.CreatedAt = time.Now()
	m.batches[b.ID] = *b
	return nil
}

func (m *memoryBatchRepo) GetByID(_ context.Context, id, _ uuid.UUID) (*domain.ImportBatch, error) {
	b, ok := m.batches[id]
	if !ok {
		return nil, domain.ErrImportBatchNotFound
	}
	return &b, nil
}

func (m *memoryBatchRepo) ListByUser(_ context.Context, userID uuid.UUID, accountID *uuid.UUID, _, _ int) ([]domain.ImportBatch, error) {
	var out []domain.ImportBatch
	for _, b := range m.batches {
		if b.UserID == userID && (accountID == nil || b.AccountID == *accountID) {
			out = append(out, b)
		}
	}
	return out, nil
}

func (m *memoryBatchRepo) UpdateStatus(_ context.Context, id, _ uuid.UUID, status domain.ImportBatchStatus, errMessage *string) error {
	b, ok := m.batches[id]
	if !ok {
		return domain.ErrImportBatchNotFound
	}
	b.Status = status
	b.ErrorMessage = errMessage
	m.batches[id] = b
	return nil
}

func (m *memoryBatchRepo) GetLatestByFileHash(_ context.Context, userID, accountID uuid.UUID, fileHash string) (*domain.ImportBatch, error) {
	for _, b := range m.batches {
		if b.UserID == userID && b.AccountID == accountID && b.FileHashSHA256 == fileHash {
			return &b, nil
		}
	}
	return nil, nil
}

type memoryTxRepo struct {
	txs []domain.Transaction
}

func (m *memoryTxRepo) Create(_ context.Context, tx *domain.Transaction) error {
	if tx.ID == uuid.Nil {
		tx.ID = uuid.New()
	}
	m.txs = append(m.txs, *tx)
	return nil
}

func (m *memoryTxRepo) GetByID(_ context.Context, id, userID uuid.UUID) (*domain.Transaction, error) {
	for i := range m.txs {
		if m.txs[i].ID == id && m.txs[i].UserID == userID && !m.txs[i].IsDeleted {
			return &m.txs[i], nil
		}
	}
	return nil, nil
}

func (m *memoryTxRepo) List(_ context.Context, userID uuid.UUID, _ domain.TransactionFilter) ([]domain.Transaction, int, error) {
	var out []domain.Transaction
	for i := range m.txs {
		if m.txs[i].UserID == userID && !m.txs[i].IsDeleted {
			out = append(out, m.txs[i])
		}
	}
	return out, len(out), nil
}

func (m *memoryTxRepo) Update(_ context.Context, _ *domain.Transaction, _ *time.Time) error {
	return nil
}

func (m *memoryTxRepo) SoftDelete(_ context.Context, id, userID uuid.UUID) error {
	for i := range m.txs {
		if m.txs[i].ID == id && m.txs[i].UserID == userID {
			m.txs[i].IsDeleted = true
			return nil
		}
	}
	return nil
}

func (m *memoryTxRepo) GetStats(_ context.Context, _ uuid.UUID, _ domain.StatsPeriod) (*domain.TransactionStats, error) {
	return nil, nil
}

func (m *memoryTxRepo) GetMonthlyExpense(_ context.Context, _ uuid.UUID, _, _ int) (float64, error) {
	return 0, nil
}

func (m *memoryTxRepo) GetCategoryExpensesByPeriod(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]domain.CategoryExpenseSummary, error) {
	return nil, nil
}

func (m *memoryTxRepo) GetCategoryIncomesByPeriod(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]domain.CategoryExpenseSummary, error) {
	return nil, nil
}

func (m *memoryTxRepo) GetCategorySavingsByPeriod(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]domain.CategoryExpenseSummary, error) {
	return nil, nil
}

func (m *memoryTxRepo) GetDailyTransactionCount(_ context.Context, _ uuid.UUID, _, _ time.Time) (int, error) {
	return 0, nil
}

func (m *memoryTxRepo) GetRecentByUser(_ context.Context, _ uuid.UUID, _ int) ([]domain.Transaction, error) {
	return nil, nil
}

func (m *memoryTxRepo) GetTimeSeries(_ context.Context, _ uuid.UUID, _, _ time.Time, _ string) ([]domain.TimeSeriesPoint, error) {
	return nil, nil
}

func (m *memoryTxRepo) GetCategoryTimeSeries(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]domain.CategoryTimeSeriesRow, error) {
	return nil, nil
}

func (m *memoryTxRepo) HasTransactions(_ context.Context, _ uuid.UUID) (bool, error) {
	return len(m.txs) > 0, nil
}

func (m *memoryTxRepo) BatchInsert(_ context.Context, txs []domain.Transaction) error {
	for i := range txs {
		tx := txs[i]
		if tx.ID == uuid.Nil {
			tx.ID = uuid.New()
		}
		m.txs = append(m.txs, tx)
	}
	return nil
}

func (m *memoryTxRepo) ListExistingFingerprints(_ context.Context, userID uuid.UUID, hashes []string) (map[string]bool, error) {
	out := make(map[string]bool)
	for i := range m.txs {
		if m.txs[i].UserID == userID && !m.txs[i].IsDeleted && m.txs[i].FingerprintHash != nil {
			for _, h := range hashes {
				if *m.txs[i].FingerprintHash == h {
					out[h] = true
				}
			}
		}
	}
	return out, nil
}

func (m *memoryTxRepo) ListByBatchID(_ context.Context, batchID, userID uuid.UUID) ([]domain.Transaction, error) {
	var out []domain.Transaction
	for i := range m.txs {
		if m.txs[i].UserID == userID && m.txs[i].ImportBatchID != nil && *m.txs[i].ImportBatchID == batchID && !m.txs[i].IsDeleted {
			out = append(out, m.txs[i])
		}
	}
	return out, nil
}

func (m *memoryTxRepo) SoftDeleteByBatchID(_ context.Context, batchID, userID uuid.UUID) (int64, error) {
	var count int64
	for i := range m.txs {
		if m.txs[i].UserID == userID && m.txs[i].ImportBatchID != nil && *m.txs[i].ImportBatchID == batchID && !m.txs[i].IsDeleted {
			m.txs[i].IsDeleted = true
			count++
		}
	}
	return count, nil
}

func (m *memoryTxRepo) UpdateTransferPair(_ context.Context, txID, pairID uuid.UUID, txType domain.TransactionType) error {
	for i := range m.txs {
		if m.txs[i].ID == txID {
			m.txs[i].TransferPairID = &pairID
			m.txs[i].Type = txType
			return nil
		}
	}
	return nil
}

type memoryTxRunner struct{}

func (m *memoryTxRunner) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (m *memoryTxRunner) AdvisoryXactLock(_ context.Context, _ string) error {
	return nil
}

func TestAccountUsecase_CRUD_And_Summary(t *testing.T) {
	accRepo := newMemoryAccountRepo()
	batchRepo := newMemoryBatchRepo()
	txRepo := &memoryTxRepo{}
	txRunner := &memoryTxRunner{}

	uc := usecase.NewAccountUsecase(accRepo, batchRepo, txRepo, txRunner, nil)
	userID := uuid.New()

	acc, err := uc.CreateAccount(context.Background(), userID, domain.UpsertFinancialAccountRequest{
		Name:            "BCA Utama",
		AccountType:     domain.AccountTypeBank,
		InstitutionCode: "bca",
		InitialBalance:  1000000,
	})
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}
	if acc.CurrentBalance != 1000000 {
		t.Errorf("current balance = %v, want 1000000", acc.CurrentBalance)
	}

	summary, err := uc.GetSummary(context.Background(), userID)
	if err != nil {
		t.Fatalf("failed to get summary: %v", err)
	}
	if summary.TotalNetWorthIDR != 1000000 {
		t.Errorf("net worth = %v, want 1000000", summary.TotalNetWorthIDR)
	}
	if summary.AccountCount != 1 {
		t.Errorf("account count = %v, want 1", summary.AccountCount)
	}
}

func TestAccountUsecase_ImportStatement_And_Rollback(t *testing.T) {
	accRepo := newMemoryAccountRepo()
	batchRepo := newMemoryBatchRepo()
	txRepo := &memoryTxRepo{}
	txRunner := &memoryTxRunner{}

	uc := usecase.NewAccountUsecase(accRepo, batchRepo, txRepo, txRunner, nil)
	userID := uuid.New()

	acc, err := uc.CreateAccount(context.Background(), userID, domain.UpsertFinancialAccountRequest{
		Name:            "BCA Rekening",
		AccountType:     domain.AccountTypeBank,
		InstitutionCode: "bca",
		InitialBalance:  1000000,
	})
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	csvData := `"TANGGAL","KETERANGAN","CBG","MUTASI","SALDO"
"01/10/2026","TRSF E-BANKING DB 2010/WS95011 QRIS SHOPEEPAY","0000","50.000,00 DB","950.000,00"
"02/10/2026","TRANSFER DR ANGGA GAJI PROYEK","0000","5.000.000,00 CR","5.950.000,00"
`

	// 1. First Import
	batch1, err := uc.ImportStatement(
		context.Background(),
		userID, acc.ID,
		"mutasi_bca.csv",
		int64(len(csvData)),
		"hash123",
		strings.NewReader(csvData),
		"bca",
	)
	if err != nil {
		t.Fatalf("first import failed: %v", err)
	}
	if batch1.ImportedRows != 2 || batch1.SkippedDuplicates != 0 {
		t.Errorf("imported=%d, skipped=%d; want 2, 0", batch1.ImportedRows, batch1.SkippedDuplicates)
	}

	updatedAcc, _ := uc.GetAccount(context.Background(), acc.ID, userID)
	// 1000000 - 50000 + 5000000 = 5950000
	if updatedAcc.CurrentBalance != 5950000 {
		t.Errorf("balance after import = %v, want 5950000", updatedAcc.CurrentBalance)
	}

	// 2. Second Import with duplicate file -> 0 imported, 2 skipped
	batch2, err := uc.ImportStatement(
		context.Background(),
		userID, acc.ID,
		"mutasi_bca_copy.csv",
		int64(len(csvData)),
		"hash123_copy",
		strings.NewReader(csvData),
		"bca",
	)
	if err != nil {
		t.Fatalf("second import failed: %v", err)
	}
	if batch2.ImportedRows != 0 || batch2.SkippedDuplicates != 2 {
		t.Errorf("re-import: imported=%d, skipped=%d; want 0, 2", batch2.ImportedRows, batch2.SkippedDuplicates)
	}

	updatedAcc2, _ := uc.GetAccount(context.Background(), acc.ID, userID)
	if updatedAcc2.CurrentBalance != 5950000 {
		t.Errorf("balance after duplicate import = %v, want 5950000", updatedAcc2.CurrentBalance)
	}

	// 3. Rollback First Batch -> Reverts balance to 1000000
	err = uc.RollbackBatch(context.Background(), userID, batch1.ID)
	if err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	rolledBackAcc, _ := uc.GetAccount(context.Background(), acc.ID, userID)
	if rolledBackAcc.CurrentBalance != 1000000 {
		t.Errorf("balance after rollback = %v, want 1000000", rolledBackAcc.CurrentBalance)
	}

	// 4. Rollback again -> should error
	err = uc.RollbackBatch(context.Background(), userID, batch1.ID)
	if err != domain.ErrImportBatchAlreadyRolledBack {
		t.Errorf("expected ErrImportBatchAlreadyRolledBack, got %v", err)
	}
}
