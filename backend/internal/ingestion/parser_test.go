package ingestion_test

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/ingestion"
)

type dummyParser struct {
	code string
}

func (d *dummyParser) InstitutionCode() string { return d.code }
func (d *dummyParser) CanParse(filename string, header []byte) bool {
	return strings.Contains(filename, d.code) || strings.Contains(string(header), d.code)
}
func (d *dummyParser) Parse(ctx context.Context, r io.Reader) ([]ingestion.RawStatementRecord, error) {
	return []ingestion.RawStatementRecord{
		{
			Date:           time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC),
			RawDescription: "Dummy transaction",
			Amount:         50000,
			Type:           domain.TransactionExpense,
		},
	}, nil
}

func TestParserRegistry(t *testing.T) {
	reg := ingestion.NewParserRegistry()

	bca := &dummyParser{code: "bca"}
	gopay := &dummyParser{code: "gopay"}

	reg.Register(bca)
	reg.Register(gopay)

	p, ok := reg.Get("bca")
	if !ok || p.InstitutionCode() != "bca" {
		t.Fatalf("expected bca parser, got %v (found: %v)", p, ok)
	}

	p2, ok2 := reg.Get("mandiri")
	if ok2 || p2 != nil {
		t.Fatalf("expected mandiri not found, got %v", p2)
	}

	detected := reg.Detect("mutasi_bca_oktober.csv", []byte("HEADER BCA"))
	if detected == nil || detected.InstitutionCode() != "bca" {
		t.Fatalf("expected detected bca, got %v", detected)
	}

	if len(reg.List()) != 2 {
		t.Fatalf("expected 2 registered parsers, got %d", len(reg.List()))
	}
}
