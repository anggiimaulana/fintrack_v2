package adapters

import (
	"context"
	"io"
	"math"
	"strings"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/ingestion"
)

type DANAParser struct{}

func NewDANAParser() *DANAParser {
	return &DANAParser{}
}

func (p *DANAParser) InstitutionCode() string {
	return "dana"
}

func (p *DANAParser) CanParse(filename string, header []byte) bool {
	lowerName := strings.ToLower(filename)
	lowerHeader := strings.ToLower(string(header))
	return strings.Contains(lowerName, "dana") ||
		(strings.Contains(lowerHeader, "dana") && strings.Contains(lowerHeader, "balance"))
}

func (p *DANAParser) Parse(ctx context.Context, r io.Reader) ([]ingestion.RawStatementRecord, error) {
	rows, err := ReadCSVRecords(r)
	if err != nil {
		return nil, err
	}

	var results []ingestion.RawStatementRecord
	for _, row := range rows {
		if len(row) < 4 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(row[0]), "date") {
			continue
		}

		t, ok := ParseFlexibleDate(row[0])
		if !ok {
			continue
		}

		desc := strings.TrimSpace(row[1])
		typeCol := strings.ToLower(strings.TrimSpace(row[2]))
		amt, ok := ParseIndonesianAmount(row[3])
		if !ok || amt == 0 {
			continue
		}

		txType := domain.TransactionExpense
		if strings.Contains(typeCol, "income") || strings.Contains(typeCol, "masuk") || strings.Contains(typeCol, "top up") {
			txType = domain.TransactionIncome
		}

		var endingBal *float64
		if len(row) >= 5 {
			if bal, ok := ParseIndonesianAmount(row[4]); ok {
				endingBal = &bal
			}
		}

		results = append(results, ingestion.RawStatementRecord{
			Date:           t,
			RawDescription: desc,
			Amount:         math.Abs(amt),
			Type:           txType,
			EndingBalance:  endingBal,
		})
	}

	return results, nil
}
