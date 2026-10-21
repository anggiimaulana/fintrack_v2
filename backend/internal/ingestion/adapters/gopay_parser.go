package adapters

import (
	"context"
	"io"
	"math"
	"strings"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/ingestion"
)

type GoPayParser struct{}

func NewGoPayParser() *GoPayParser {
	return &GoPayParser{}
}

func (p *GoPayParser) InstitutionCode() string {
	return "gopay"
}

func (p *GoPayParser) CanParse(filename string, header []byte) bool {
	lowerName := strings.ToLower(filename)
	lowerHeader := strings.ToLower(string(header))
	return strings.Contains(lowerName, "gopay") || strings.Contains(lowerName, "gojek") ||
		(strings.Contains(lowerHeader, "transaction date") && strings.Contains(lowerHeader, "amount (idr)"))
}

func (p *GoPayParser) Parse(ctx context.Context, r io.Reader) ([]ingestion.RawStatementRecord, error) {
	rows, err := ReadCSVRecords(r)
	if err != nil {
		return nil, err
	}

	var results []ingestion.RawStatementRecord
	for _, row := range rows {
		if len(row) < 5 {
			continue
		}
		if strings.Contains(strings.ToLower(row[0]), "transaction date") {
			continue
		}

		t, ok := ParseFlexibleDate(row[0])
		if !ok {
			continue
		}

		ref := strings.TrimSpace(row[1])
		typeCol := strings.ToLower(strings.TrimSpace(row[2]))
		desc := strings.TrimSpace(row[3])
		amt, ok := ParseIndonesianAmount(row[4])
		if !ok || amt == 0 {
			continue
		}

		txType := domain.TransactionExpense
		if amt > 0 && (strings.Contains(typeCol, "top up") || strings.Contains(typeCol, "transfer in") || strings.Contains(typeCol, "receive")) {
			txType = domain.TransactionIncome
		}

		results = append(results, ingestion.RawStatementRecord{
			Date:            t,
			RawDescription:  desc,
			Amount:          math.Abs(amt),
			Type:            txType,
			ReferenceNumber: ref,
		})
	}

	return results, nil
}
