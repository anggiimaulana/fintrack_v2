package adapters

import (
	"context"
	"io"
	"math"
	"strings"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/ingestion"
)

type MandiriParser struct{}

func NewMandiriParser() *MandiriParser {
	return &MandiriParser{}
}

func (p *MandiriParser) InstitutionCode() string {
	return "mandiri"
}

func (p *MandiriParser) CanParse(filename string, header []byte) bool {
	lowerName := strings.ToLower(filename)
	lowerHeader := strings.ToLower(string(header))
	return strings.Contains(lowerName, "mandiri") || strings.Contains(lowerName, "livin") ||
		(strings.Contains(lowerHeader, "debit") && strings.Contains(lowerHeader, "kredit") && strings.Contains(lowerHeader, "keterangan"))
}

func (p *MandiriParser) Parse(ctx context.Context, r io.Reader) ([]ingestion.RawStatementRecord, error) {
	rows, err := ReadCSVRecords(r)
	if err != nil {
		return nil, err
	}

	var results []ingestion.RawStatementRecord
	for _, row := range rows {
		if len(row) < 4 {
			continue
		}
		if strings.Contains(strings.ToLower(row[0]), "tanggal") {
			continue
		}

		t, ok := ParseFlexibleDate(row[0])
		if !ok {
			continue
		}

		desc := strings.TrimSpace(row[1])
		debitStr := strings.TrimSpace(row[2])
		creditStr := strings.TrimSpace(row[3])

		var amount float64
		var txType domain.TransactionType

		if debitStr != "" {
			if amt, ok := ParseIndonesianAmount(debitStr); ok && amt > 0 {
				amount = amt
				txType = domain.TransactionExpense
			}
		}
		if amount == 0 && creditStr != "" {
			if amt, ok := ParseIndonesianAmount(creditStr); ok && amt > 0 {
				amount = amt
				txType = domain.TransactionIncome
			}
		}

		if amount == 0 {
			continue
		}

		var endingBalance *float64
		if len(row) >= 5 {
			if bal, ok := ParseIndonesianAmount(row[4]); ok {
				endingBalance = &bal
			}
		}

		results = append(results, ingestion.RawStatementRecord{
			Date:           t,
			RawDescription: desc,
			Amount:         math.Abs(amount),
			Type:           txType,
			EndingBalance:  endingBalance,
		})
	}

	return results, nil
}
