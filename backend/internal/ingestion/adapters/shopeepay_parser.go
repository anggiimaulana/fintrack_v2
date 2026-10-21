package adapters

import (
	"context"
	"io"
	"math"
	"strings"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/ingestion"
)

type ShopeePayParser struct{}

func NewShopeePayParser() *ShopeePayParser {
	return &ShopeePayParser{}
}

func (p *ShopeePayParser) InstitutionCode() string {
	return "shopeepay"
}

func (p *ShopeePayParser) CanParse(filename string, header []byte) bool {
	lowerName := strings.ToLower(filename)
	lowerHeader := strings.ToLower(string(header))
	return strings.Contains(lowerName, "shopee") ||
		(strings.Contains(lowerHeader, "reference id") && strings.Contains(lowerHeader, "shopee"))
}

func (p *ShopeePayParser) Parse(ctx context.Context, r io.Reader) ([]ingestion.RawStatementRecord, error) {
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

		ref := strings.TrimSpace(row[1])
		desc := strings.TrimSpace(row[2])
		rawAmtStr := strings.TrimSpace(row[3])
		isExplicitNegative := strings.HasPrefix(rawAmtStr, "-")

		amt, ok := ParseIndonesianAmount(rawAmtStr)
		if !ok || amt == 0 {
			continue
		}

		txType := domain.TransactionExpense
		if !isExplicitNegative && (amt > 0 && (strings.Contains(strings.ToLower(desc), "top up") || strings.Contains(strings.ToLower(desc), "masuk") || strings.Contains(strings.ToLower(desc), "cashback"))) {
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
