package adapters

import (
	"context"
	"io"
	"math"
	"strings"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/ingestion"
)

type SeaBankParser struct{}

func NewSeaBankParser() *SeaBankParser {
	return &SeaBankParser{}
}

func (p *SeaBankParser) InstitutionCode() string {
	return "seabank"
}

func (p *SeaBankParser) CanParse(filename string, header []byte) bool {
	lowerName := strings.ToLower(filename)
	lowerHeader := strings.ToLower(string(header))
	return strings.Contains(lowerName, "seabank") ||
		(strings.Contains(lowerHeader, "waktu transaksi") && strings.Contains(lowerHeader, "jenis transaksi"))
}

func (p *SeaBankParser) Parse(ctx context.Context, r io.Reader) ([]ingestion.RawStatementRecord, error) {
	rows, err := ReadCSVRecords(r)
	if err != nil {
		return nil, err
	}

	var results []ingestion.RawStatementRecord
	for _, row := range rows {
		if len(row) < 3 {
			continue
		}
		if strings.Contains(strings.ToLower(row[0]), "waktu") {
			continue
		}

		t, ok := ParseFlexibleDate(row[0])
		if !ok {
			continue
		}

		jenis := strings.TrimSpace(row[1])
		amt, ok := ParseIndonesianAmount(row[2])
		if !ok || amt == 0 {
			continue
		}

		txType := domain.TransactionExpense
		lowerJenis := strings.ToLower(jenis)
		if strings.Contains(lowerJenis, "masuk") || strings.Contains(lowerJenis, "bunga") || strings.Contains(lowerJenis, "inflow") {
			txType = domain.TransactionIncome
		}

		ref := ""
		if len(row) >= 5 {
			ref = strings.TrimSpace(row[4])
		}

		results = append(results, ingestion.RawStatementRecord{
			Date:            t,
			RawDescription:  jenis,
			Amount:          math.Abs(amt),
			Type:            txType,
			ReferenceNumber: ref,
		})
	}

	return results, nil
}
