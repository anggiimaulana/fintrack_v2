package ingestion_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/ingestion"
)

func TestGenerateFingerprint(t *testing.T) {
	userID := uuid.New()
	accountID := uuid.New()
	date := time.Date(2026, 10, 20, 14, 30, 0, 0, time.UTC)
	amount := 75000.00

	h1 := ingestion.GenerateFingerprint(userID, accountID, date, amount, "TRSF E-BANKING DB QRIS SHOPEEPAY")
	h2 := ingestion.GenerateFingerprint(userID, accountID, date, amount, "  trsf e-banking db qris shopeepay  ")

	if h1 == "" {
		t.Fatal("expected non-empty hash")
	}
	if h1 != h2 {
		t.Errorf("expected hashes to match for normalized text: %q vs %q", h1, h2)
	}

	hDiffAmount := ingestion.GenerateFingerprint(userID, accountID, date, 75000.50, "TRSF E-BANKING DB QRIS SHOPEEPAY")
	if h1 == hDiffAmount {
		t.Errorf("expected different hash for different amount")
	}

	hDiffDate := ingestion.GenerateFingerprint(userID, accountID, date.AddDate(0, 0, 1), amount, "TRSF E-BANKING DB QRIS SHOPEEPAY")
	if h1 == hDiffDate {
		t.Errorf("expected different hash for different date")
	}
}

func TestNormalizer_Stage1Regex(t *testing.T) {
	normalizer := ingestion.NewNormalizer()

	cases := []struct {
		raw           string
		cleanMerchant string
	}{
		{"TRSF E-BANKING DB 2010/WS95011 QRIS SHOPEEPAY", "ShopeePay"},
		{"M-BCA 20/10 WSID:88123 TOKO KOPI TUKU", "Toko Kopi Tuku"},
		{"BI-FAST TRF KE KANTONG JAGO", "Kantong Jago"},
		{"TRF DR 014 ANGGA TO TOKOPEDIA", "Tokopedia"},
	}

	for _, tc := range cases {
		res := normalizer.NormalizeStage1(tc.raw)
		if res.CleanMerchant != tc.cleanMerchant {
			t.Errorf("NormalizeStage1(%q) CleanMerchant = %q, want %q", tc.raw, res.CleanMerchant, tc.cleanMerchant)
		}
	}
}

func TestPairInternalTransfers(t *testing.T) {
	userID := uuid.New()
	accountA := uuid.New()
	accountB := uuid.New()

	now := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)

	tx1 := domain.Transaction{
		ID:              uuid.New(),
		UserID:          userID,
		AccountID:       &accountA,
		Type:            domain.TransactionExpense,
		Amount:          500000,
		AmountBase:      500000,
		TransactionDate: now,
		Description:     new("TRF KE MANDIRI"),
	}

	tx2 := domain.Transaction{
		ID:              uuid.New(),
		UserID:          userID,
		AccountID:       &accountB,
		Type:            domain.TransactionIncome,
		Amount:          500000,
		AmountBase:      500000,
		TransactionDate: now.Add(15 * time.Minute),
		Description:     new("TRF DR BCA"),
	}

	tx3Unrelated := domain.Transaction{
		ID:              uuid.New(),
		UserID:          userID,
		AccountID:       &accountA,
		Type:            domain.TransactionExpense,
		Amount:          25000,
		AmountBase:      25000,
		TransactionDate: now,
		Description:     new("KOPI TUKU"),
	}

	txs := []domain.Transaction{tx1, tx2, tx3Unrelated}

	pairs := ingestion.PairInternalTransfers(txs)
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}

	pair := pairs[0]
	if pair.OutboundTxID != tx1.ID || pair.InboundTxID != tx2.ID {
		t.Errorf("pair IDs mismatch: out=%v, in=%v", pair.OutboundTxID, pair.InboundTxID)
	}
}
