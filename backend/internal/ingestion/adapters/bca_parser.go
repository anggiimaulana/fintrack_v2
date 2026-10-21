package adapters

import (
	"context"
	"io"
	"math"
	"strings"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/ingestion"
)

type BCAParser struct{}

func NewBCAParser() *BCAParser {
	return &BCAParser{}
}

func (p *BCAParser) InstitutionCode() string {
	return "bca"
}

func (p *BCAParser) CanParse(filename string, header []byte) bool {
	lowerName := strings.ToLower(filename)
	lowerHeader := strings.ToLower(string(header))
	return strings.Contains(lowerName, "bca") ||
		(strings.Contains(lowerHeader, "tanggal") && strings.Contains(lowerHeader, "mutasi") && strings.Contains(lowerHeader, "saldo"))
}

func (p *BCAParser) Parse(ctx context.Context, r io.Reader) ([]ingestion.RawStatementRecord, error) {
	rows, err := ReadCSVRecords(r)
	if err != nil {
		return nil, err
	}

	var results []ingestion.RawStatementRecord
	for _, row := range rows {
		if len(row) < 4 {
			continue
		}
		// Skip header
		if strings.EqualFold(strings.TrimSpace(row[0]), "tanggal") {
			continue
		}

		dateStr := strings.TrimSpace(row[0])
		t, ok := ParseFlexibleDate(dateStr)
		if !ok {
			continue
		}

		desc := strings.TrimSpace(row[1])
		mutasiStr := strings.TrimSpace(row[3])

		isCredit := strings.HasSuffix(strings.ToUpper(mutasiStr), "CR")
		cleanMutasi := strings.TrimSuffix(strings.TrimSuffix(strings.ToUpper(mutasiStr), "CR"), "DB")
		cleanMutasi = strings.TrimSpace(cleanMutasi)

		amount, validAmt := ParseIndonesianAmount(cleanMutasi)
		if !validAmt || amount == 0 {
			continue
		}
		amount = math.Abs(amount)

		txType := domain.TransactionExpense
		if isCredit {
			txType = domain.TransactionIncome
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
			Amount:         amount,
			Type:           txType,
			EndingBalance:  endingBalance,
		})
	}

	return results, nil
}
