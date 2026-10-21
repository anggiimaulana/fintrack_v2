package adapters

import (
	"context"
	"io"
	"math"
	"strings"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/ingestion"
)

type JagoParser struct{}

func NewJagoParser() *JagoParser {
	return &JagoParser{}
}

func (p *JagoParser) InstitutionCode() string {
	return "jago"
}

func (p *JagoParser) CanParse(filename string, header []byte) bool {
	lowerName := strings.ToLower(filename)
	lowerHeader := strings.ToLower(string(header))
	return strings.Contains(lowerName, "jago") ||
		(strings.Contains(lowerHeader, "kantong") && strings.Contains(lowerHeader, "tipe transaksi"))
}

func (p *JagoParser) Parse(ctx context.Context, r io.Reader) ([]ingestion.RawStatementRecord, error) {
	rows, err := ReadCSVRecords(r)
	if err != nil {
		return nil, err
	}

	var results []ingestion.RawStatementRecord
	for _, row := range rows {
		if len(row) < 3 {
			continue
		}
		if strings.Contains(strings.ToLower(row[0]), "tanggal") {
			continue
		}

		t, ok := ParseFlexibleDate(row[0])
		if !ok {
			continue
		}

		typeStr := strings.ToLower(strings.TrimSpace(row[1]))
		amt, ok := ParseIndonesianAmount(row[2])
		if !ok || amt == 0 {
			continue
		}

		txType := domain.TransactionExpense
		if strings.Contains(typeStr, "masuk") || strings.Contains(typeStr, "in") || amt > 0 && strings.Contains(typeStr, "income") {
			txType = domain.TransactionIncome
		}

		desc := "Transaksi Jago"
		if len(row) >= 4 && strings.TrimSpace(row[3]) != "" {
			desc = strings.TrimSpace(row[3])
		}
		if len(row) >= 5 && strings.TrimSpace(row[4]) != "" {
			desc = desc + " (" + strings.TrimSpace(row[4]) + ")"
		}

		results = append(results, ingestion.RawStatementRecord{
			Date:           t,
			RawDescription: desc,
			Amount:         math.Abs(amt),
			Type:           txType,
		})
	}

	return results, nil
}
