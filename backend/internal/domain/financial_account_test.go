package domain_test

import (
	"testing"

	"github.com/fintrack/backend/internal/domain"
)

func TestFinancialAccountType_Valid(t *testing.T) {
	cases := []struct {
		kind  domain.FinancialAccountType
		valid bool
	}{
		{domain.AccountTypeBank, true},
		{domain.AccountTypeEwallet, true},
		{domain.AccountTypeCash, true},
		{domain.AccountTypeInvestment, true},
		{domain.FinancialAccountType("crypto"), false},
		{domain.FinancialAccountType(""), false},
	}

	for _, tc := range cases {
		if got := tc.kind.Valid(); got != tc.valid {
			t.Errorf("FinancialAccountType(%q).Valid() = %v, want %v", tc.kind, got, tc.valid)
		}
	}
}

func TestImportBatchStatus_Valid(t *testing.T) {
	cases := []struct {
		status domain.ImportBatchStatus
		valid  bool
	}{
		{domain.BatchStatusProcessing, true},
		{domain.BatchStatusCompleted, true},
		{domain.BatchStatusFailed, true},
		{domain.BatchStatusRolledBack, true},
		{domain.ImportBatchStatus("pending"), false},
		{domain.ImportBatchStatus(""), false},
	}

	for _, tc := range cases {
		if got := tc.status.Valid(); got != tc.valid {
			t.Errorf("ImportBatchStatus(%q).Valid() = %v, want %v", tc.status, got, tc.valid)
		}
	}
}

func TestTransactionType_Transfer(t *testing.T) {
	if domain.TransactionTransfer != "transfer" {
		t.Errorf("TransactionTransfer = %q, want 'transfer'", domain.TransactionTransfer)
	}
}

func TestTransactionSource_Statement(t *testing.T) {
	if domain.SourceStatement != "statement" {
		t.Errorf("SourceStatement = %q, want 'statement'", domain.SourceStatement)
	}
}
