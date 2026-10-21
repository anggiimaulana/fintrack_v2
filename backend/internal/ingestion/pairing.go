package ingestion

import (
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/fintrack/backend/internal/domain"
)

// TransferPair represents two matching transactions across user accounts.
type TransferPair struct {
	OutboundTxID uuid.UUID
	InboundTxID  uuid.UUID
}

// PairInternalTransfers detects matching transfers between user accounts within 36 hours.
func PairInternalTransfers(txs []domain.Transaction) []TransferPair {
	var pairs []TransferPair
	used := make(map[uuid.UUID]bool)
	maxTimeDiff := 36 * time.Hour

	for i := range txs {
		t1 := txs[i]
		if used[t1.ID] || t1.AccountID == nil {
			continue
		}

		for j := i + 1; j < len(txs); j++ {
			t2 := txs[j]
			if used[t2.ID] || t2.AccountID == nil {
				continue
			}

			// Must belong to same user but different accounts
			if t1.UserID != t2.UserID || *t1.AccountID == *t2.AccountID {
				continue
			}

			// Amounts must match
			if math.Abs(t1.AmountBase-t2.AmountBase) > 0.01 {
				continue
			}

			// Date difference within window
			diff := t1.TransactionDate.Sub(t2.TransactionDate)
			if diff < 0 {
				diff = -diff
			}
			if diff > maxTimeDiff {
				continue
			}

			// Must be opposite directions (one expense, one income)
			if t1.Type == domain.TransactionExpense && t2.Type == domain.TransactionIncome {
				pairs = append(pairs, TransferPair{
					OutboundTxID: t1.ID,
					InboundTxID:  t2.ID,
				})
				used[t1.ID] = true
				used[t2.ID] = true
				break
			} else if t1.Type == domain.TransactionIncome && t2.Type == domain.TransactionExpense {
				pairs = append(pairs, TransferPair{
					OutboundTxID: t2.ID,
					InboundTxID:  t1.ID,
				})
				used[t1.ID] = true
				used[t2.ID] = true
				break
			}
		}
	}

	return pairs
}
