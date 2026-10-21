package usecase

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/ingestion"
	"github.com/fintrack/backend/internal/ingestion/adapters"
)

type AccountUsecase struct {
	accountRepo domain.FinancialAccountRepository
	batchRepo   domain.ImportBatchRepository
	txRepo      domain.TransactionRepository
	registry    *ingestion.ParserRegistry
	normalizer  *ingestion.Normalizer
	txRunner    domain.TxRunner
}

func NewAccountUsecase(
	accountRepo domain.FinancialAccountRepository,
	batchRepo domain.ImportBatchRepository,
	txRepo domain.TransactionRepository,
	txRunner domain.TxRunner,
	normalizer *ingestion.Normalizer,
	registry ...*ingestion.ParserRegistry,
) *AccountUsecase {
	var reg *ingestion.ParserRegistry
	if len(registry) > 0 && registry[0] != nil {
		reg = registry[0]
	} else {
		reg = adapters.NewDefaultRegistry()
	}

	if normalizer == nil {
		normalizer = ingestion.NewNormalizer()
	}

	return &AccountUsecase{
		accountRepo: accountRepo,
		batchRepo:   batchRepo,
		txRepo:      txRepo,
		registry:    reg,
		normalizer:  normalizer,
		txRunner:    txRunner,
	}
}

func (u *AccountUsecase) CreateAccount(ctx context.Context, userID uuid.UUID, req domain.UpsertFinancialAccountRequest) (*domain.FinancialAccount, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, domain.ErrFinancialAccountNameEmpty
	}
	if strings.TrimSpace(req.InstitutionCode) == "" {
		return nil, domain.ErrFinancialAccountInstEmpty
	}
	if !req.AccountType.Valid() {
		return nil, domain.ErrFinancialAccountInvalidType
	}

	curr := strings.TrimSpace(req.CurrencyCode)
	if curr == "" {
		curr = "IDR"
	}
	color := strings.TrimSpace(req.ColorHex)
	if color == "" {
		color = "#2563EB"
	}
	icon := strings.TrimSpace(req.IconKey)
	if icon == "" {
		icon = "wallet"
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	account := &domain.FinancialAccount{
		UserID:            userID,
		Name:              strings.TrimSpace(req.Name),
		AccountType:       req.AccountType,
		InstitutionCode:   strings.ToLower(strings.TrimSpace(req.InstitutionCode)),
		AccountNumberMask: req.AccountNumberMask,
		CurrencyCode:      curr,
		CurrentBalance:    req.InitialBalance,
		InitialBalance:    req.InitialBalance,
		ColorHex:          color,
		IconKey:           icon,
		IsActive:          isActive,
	}

	if err := u.accountRepo.Create(ctx, account); err != nil {
		return nil, err
	}
	return account, nil
}

func (u *AccountUsecase) GetAccount(ctx context.Context, id, userID uuid.UUID) (*domain.FinancialAccount, error) {
	return u.accountRepo.GetByID(ctx, id, userID)
}

func (u *AccountUsecase) ListAccounts(ctx context.Context, userID uuid.UUID, onlyActive bool) ([]domain.FinancialAccount, error) {
	return u.accountRepo.ListByUser(ctx, userID, onlyActive)
}

func (u *AccountUsecase) UpdateAccount(ctx context.Context, id, userID uuid.UUID, req domain.UpsertFinancialAccountRequest) (*domain.FinancialAccount, error) {
	existing, err := u.accountRepo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(req.Name) != "" {
		existing.Name = strings.TrimSpace(req.Name)
	}
	if req.AccountType.Valid() {
		existing.AccountType = req.AccountType
	}
	if strings.TrimSpace(req.InstitutionCode) != "" {
		existing.InstitutionCode = strings.ToLower(strings.TrimSpace(req.InstitutionCode))
	}
	if req.AccountNumberMask != nil {
		existing.AccountNumberMask = req.AccountNumberMask
	}
	if strings.TrimSpace(req.ColorHex) != "" {
		existing.ColorHex = strings.TrimSpace(req.ColorHex)
	}
	if strings.TrimSpace(req.IconKey) != "" {
		existing.IconKey = strings.TrimSpace(req.IconKey)
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	if err := u.accountRepo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (u *AccountUsecase) DeleteAccount(ctx context.Context, id, userID uuid.UUID) error {
	return u.accountRepo.Delete(ctx, id, userID)
}

func (u *AccountUsecase) GetSummary(ctx context.Context, userID uuid.UUID) (*domain.FinancialAccountSummary, error) {
	accounts, err := u.accountRepo.ListByUser(ctx, userID, true)
	if err != nil {
		return nil, err
	}

	summary := &domain.FinancialAccountSummary{
		Accounts:        accounts,
		AccountCount:    len(accounts),
		BreakdownByType: make(map[domain.FinancialAccountType]float64),
		BreakdownByInst: make(map[string]float64),
	}

	for _, a := range accounts {
		summary.TotalNetWorthIDR += a.CurrentBalance
		summary.BreakdownByType[a.AccountType] += a.CurrentBalance
		summary.BreakdownByInst[a.InstitutionCode] += a.CurrentBalance
	}

	return summary, nil
}

func (u *AccountUsecase) ImportStatement(
	ctx context.Context,
	userID, accountID uuid.UUID,
	fileName string,
	fileSizeBytes int64,
	fileHash string,
	r io.Reader,
	institutionHint string,
) (*domain.ImportBatch, error) {
	account, err := u.accountRepo.GetByID(ctx, accountID, userID)
	if err != nil {
		return nil, err
	}

	// 1. Select Parser
	instCode := strings.ToLower(strings.TrimSpace(institutionHint))
	if instCode == "" {
		instCode = account.InstitutionCode
	}

	parser, ok := u.registry.Get(instCode)
	if !ok {
		parser = u.registry.Detect(fileName, nil)
	}
	if parser == nil {
		return nil, fmt.Errorf("tidak ditemukan parser mutasi untuk institusi %q", instCode)
	}

	// 2. Parse Raw Records
	rawRecords, err := parser.Parse(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca berkas mutasi: %w", err)
	}

	if len(rawRecords) == 0 {
		batch := &domain.ImportBatch{
			UserID:         userID,
			AccountID:      accountID,
			FileName:       fileName,
			FileSizeBytes:  fileSizeBytes,
			FileHashSHA256: fileHash,
			TotalRows:      0,
			Status:         domain.BatchStatusCompleted,
		}
		_ = u.batchRepo.Create(ctx, batch)
		return batch, nil
	}

	// 3. Compute Fingerprints
	fingerprints := make([]string, len(rawRecords))
	for i := range rawRecords {
		rec := rawRecords[i]
		fingerprints[i] = ingestion.GenerateFingerprint(userID, accountID, rec.Date, rec.Amount, rec.RawDescription)
	}

	// 4. Query Duplicates
	existingHashes, err := u.txRepo.ListExistingFingerprints(ctx, userID, fingerprints)
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa duplikasi transaksi: %w", err)
	}

	// 5. Build New Transactions
	var newTxs []domain.Transaction
	var skippedCount int
	var netDelta float64

	for i := range rawRecords {
		rec := rawRecords[i]
		fp := fingerprints[i]

		if existingHashes[fp] {
			skippedCount++
			continue
		}

		norm := u.normalizer.Normalize(ctx, rec.RawDescription)

		tx := domain.Transaction{
			UserID:          userID,
			AccountID:       &accountID,
			FingerprintHash: &fp,
			CleanMerchant:   &norm.CleanMerchant,
			Type:            rec.Type,
			Amount:          rec.Amount,
			CurrencyCode:    account.CurrencyCode,
			ExchangeRate:    1.0,
			AmountBase:      rec.Amount,
			Description:     &rec.RawDescription,
			TransactionDate: rec.Date,
			Source:          domain.SourceStatement,
			Tags:            []string{"statement_import", account.InstitutionCode},
		}

		if rec.Type == domain.TransactionIncome {
			netDelta += rec.Amount
		} else if rec.Type == domain.TransactionExpense {
			netDelta -= rec.Amount
		}

		newTxs = append(newTxs, tx)
	}

	// 6. Atomic DB Execution
	batch := &domain.ImportBatch{
		UserID:            userID,
		AccountID:         accountID,
		FileName:          fileName,
		FileSizeBytes:     fileSizeBytes,
		FileHashSHA256:    fileHash,
		TotalRows:         len(rawRecords),
		ImportedRows:      len(newTxs),
		SkippedDuplicates: skippedCount,
		Status:            domain.BatchStatusCompleted,
	}

	runTxErr := u.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		// Lock account row for update
		lockedAccount, err := u.accountRepo.GetByIDForUpdate(txCtx, accountID, userID)
		if err != nil {
			return err
		}

		// Save import batch
		if err := u.batchRepo.Create(txCtx, batch); err != nil {
			return err
		}

		// Set batch ID on transactions
		for i := range newTxs {
			newTxs[i].ImportBatchID = &batch.ID
		}

		// Batch insert transactions
		if len(newTxs) > 0 {
			if err := u.txRepo.BatchInsert(txCtx, newTxs); err != nil {
				return err
			}
		}

		// Update running balance
		newBalance := lockedAccount.CurrentBalance + netDelta
		if err := u.accountRepo.UpdateBalance(txCtx, accountID, userID, newBalance); err != nil {
			return err
		}

		return nil
	})

	if runTxErr != nil {
		return nil, fmt.Errorf("commit mutasi transaksi: %w", runTxErr)
	}

	// 7. Post-commit: Detect internal transfers across user's active transactions
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		u.detectAndPairTransfers(bgCtx, userID)
	}()

	return batch, nil
}

func (u *AccountUsecase) detectAndPairTransfers(ctx context.Context, userID uuid.UUID) {
	// Look up recent transactions across accounts to pair transfers
	now := time.Now()
	dateFrom := now.AddDate(0, -1, 0)
	txs, _, err := u.txRepo.List(ctx, userID, domain.TransactionFilter{
		PerPage:  200,
		DateFrom: &dateFrom,
	})
	if err != nil || len(txs) < 2 {
		return
	}

	pairs := ingestion.PairInternalTransfers(txs)
	for _, pair := range pairs {
		_ = u.txRepo.UpdateTransferPair(ctx, pair.OutboundTxID, pair.InboundTxID, domain.TransactionTransfer)
		_ = u.txRepo.UpdateTransferPair(ctx, pair.InboundTxID, pair.OutboundTxID, domain.TransactionTransfer)
	}
}

func (u *AccountUsecase) RollbackBatch(ctx context.Context, userID, batchID uuid.UUID) error {
	batch, err := u.batchRepo.GetByID(ctx, batchID, userID)
	if err != nil {
		return err
	}
	if batch.Status == domain.BatchStatusRolledBack {
		return domain.ErrImportBatchAlreadyRolledBack
	}
	if batch.Status != domain.BatchStatusCompleted {
		return domain.ErrImportBatchNotCompleted
	}

	return u.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		// 1. Lock account row
		account, err := u.accountRepo.GetByIDForUpdate(txCtx, batch.AccountID, userID)
		if err != nil {
			return err
		}

		// 2. Fetch all transactions in this batch
		txs, err := u.txRepo.ListByBatchID(txCtx, batchID, userID)
		if err != nil {
			return err
		}

		// 3. Compute reverse delta
		var reverseDelta float64
		for _, tx := range txs {
			if tx.Type == domain.TransactionIncome {
				reverseDelta -= tx.AmountBase
			} else if tx.Type == domain.TransactionExpense {
				reverseDelta += tx.AmountBase
			}
		}

		// 4. Soft delete transactions
		if _, err := u.txRepo.SoftDeleteByBatchID(txCtx, batchID, userID); err != nil {
			return err
		}

		// 5. Update running balance
		newBalance := account.CurrentBalance + reverseDelta
		if err := u.accountRepo.UpdateBalance(txCtx, batch.AccountID, userID, newBalance); err != nil {
			return err
		}

		// 6. Update batch status
		return u.batchRepo.UpdateStatus(txCtx, batchID, userID, domain.BatchStatusRolledBack, nil)
	})
}

func (u *AccountUsecase) ListBatches(ctx context.Context, userID uuid.UUID, accountID *uuid.UUID, limit, offset int) ([]domain.ImportBatch, error) {
	return u.batchRepo.ListByUser(ctx, userID, accountID, limit, offset)
}
