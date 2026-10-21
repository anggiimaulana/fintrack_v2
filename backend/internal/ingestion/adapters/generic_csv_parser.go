package adapters

import (
	"context"
	"io"
	"math"
	"strings"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/ingestion"
)

type GenericCSVParser struct{}

func NewGenericCSVParser() *GenericCSVParser {
	return &GenericCSVParser{}
}

func (p *GenericCSVParser) InstitutionCode() string {
	return "generic"
}

func (p *GenericCSVParser) CanParse(filename string, header []byte) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".csv")
}

func (p *GenericCSVParser) Parse(ctx context.Context, r io.Reader) ([]ingestion.RawStatementRecord, error) {
	rows, err := ReadCSVRecords(r)
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, nil
	}

	header := rows[0]
	dateIdx, descIdx, amtIdx, typeIdx := -1, -1, -1, -1

	for i, col := range header {
		c := strings.ToLower(strings.TrimSpace(col))
		switch {
		case strings.Contains(c, "date") || strings.Contains(c, "tanggal"):
			if dateIdx == -1 {
				dateIdx = i
			}
		case strings.Contains(c, "desc") || strings.Contains(c, "keterangan") || strings.Contains(c, "uraian"):
			if descIdx == -1 {
				descIdx = i
			}
		case strings.Contains(c, "amount") || strings.Contains(c, "jumlah") || strings.Contains(c, "nominal"):
			if amtIdx == -1 {
				amtIdx = i
			}
		case strings.Contains(c, "type") || strings.Contains(c, "tipe") || strings.Contains(c, "kategori") || strings.Contains(c, "direction"):
			if typeIdx == -1 {
				typeIdx = i
			}
		}
	}

	if dateIdx == -1 || amtIdx == -1 {
		return nil, nil
	}

	var results []ingestion.RawStatementRecord
	for _, row := range rows[1:] {
		if len(row) <= dateIdx || len(row) <= amtIdx {
			continue
		}

		t, ok := ParseFlexibleDate(row[dateIdx])
		if !ok {
			continue
		}

		amt, ok := ParseIndonesianAmount(row[amtIdx])
		if !ok || amt == 0 {
			continue
		}

		desc := "Mutasi Transaksi"
		if descIdx != -1 && len(row) > descIdx && strings.TrimSpace(row[descIdx]) != "" {
			desc = strings.TrimSpace(row[descIdx])
		}

		txType := domain.TransactionExpense
		if amt > 0 {
			if typeIdx != -1 && len(row) > typeIdx {
				tStr := strings.ToLower(strings.TrimSpace(row[typeIdx]))
				if strings.Contains(tStr, "income") || strings.Contains(tStr, "masuk") || strings.Contains(tStr, "cr") {
					txType = domain.TransactionIncome
				}
			}
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
