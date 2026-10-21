package adapters_test

import (
	"context"
	"strings"
	"testing"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/ingestion/adapters"
)

func TestBCAParser(t *testing.T) {
	parser := adapters.NewBCAParser()
	if parser.InstitutionCode() != "bca" {
		t.Fatalf("expected institution code bca, got %s", parser.InstitutionCode())
	}

	csvData := `"TANGGAL","KETERANGAN","CBG","MUTASI","SALDO"
"01/10/2026","TRSF E-BANKING DB 2010/WS95011 QRIS SHOPEEPAY","0000","50.000,00 DB","1.450.000,00"
"02/10/2026","TRANSFER DR ANGGA GAJI PROYEK","0000","5.000.000,00 CR","6.450.000,00"
`
	if !parser.CanParse("mutasi_bca.csv", []byte(csvData[:40])) {
		t.Fatalf("expected CanParse true for BCA CSV")
	}

	records, err := parser.Parse(context.Background(), strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("failed to parse BCA CSV: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	// Record 1
	if records[0].Amount != 50000 || records[0].Type != domain.TransactionExpense {
		t.Errorf("record 0 mismatch: amount=%v type=%v", records[0].Amount, records[0].Type)
	}
	if records[0].Date.Day() != 1 || records[0].Date.Month() != 10 || records[0].Date.Year() != 2026 {
		t.Errorf("record 0 date mismatch: %v", records[0].Date)
	}

	// Record 2
	if records[1].Amount != 5000000 || records[1].Type != domain.TransactionIncome {
		t.Errorf("record 1 mismatch: amount=%v type=%v", records[1].Amount, records[1].Type)
	}
}

func TestMandiriParser(t *testing.T) {
	parser := adapters.NewMandiriParser()
	if parser.InstitutionCode() != "mandiri" {
		t.Fatalf("expected institution code mandiri, got %s", parser.InstitutionCode())
	}

	csvData := `Tanggal Transaksi,Keterangan,Debit,Kredit,Saldo
05/10/2026,BI-FAST TRF KE JAGO,250000.00,,4750000.00
06/10/2026,SETOR TUNAI,,1000000.00,5750000.00
`
	if !parser.CanParse("mandiri.csv", []byte(csvData[:40])) {
		t.Fatalf("expected CanParse true for Mandiri CSV")
	}

	records, err := parser.Parse(context.Background(), strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("failed to parse Mandiri CSV: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	if records[0].Amount != 250000 || records[0].Type != domain.TransactionExpense {
		t.Errorf("record 0 mismatch: %v", records[0])
	}
	if records[1].Amount != 1000000 || records[1].Type != domain.TransactionIncome {
		t.Errorf("record 1 mismatch: %v", records[1])
	}
}

func TestJagoParser(t *testing.T) {
	parser := adapters.NewJagoParser()
	if parser.InstitutionCode() != "jago" {
		t.Fatalf("expected institution code jago, got %s", parser.InstitutionCode())
	}

	csvData := `Tanggal,Tipe Transaksi,Jumlah,Penerima/Pengirim,Kantong
2026-10-10,Uang Keluar,75000,Kopi Kenangan,Jajan
2026-10-11,Uang Masuk,250000,BCA Angga,Tabungan
`
	if !parser.CanParse("jago_statement.csv", []byte(csvData[:40])) {
		t.Fatalf("expected CanParse true for Jago CSV")
	}

	records, err := parser.Parse(context.Background(), strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("failed to parse Jago CSV: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Amount != 75000 || records[0].Type != domain.TransactionExpense {
		t.Errorf("record 0 mismatch: %v", records[0])
	}
	if records[1].Amount != 250000 || records[1].Type != domain.TransactionIncome {
		t.Errorf("record 1 mismatch: %v", records[1])
	}
}

func TestSeaBankParser(t *testing.T) {
	parser := adapters.NewSeaBankParser()
	if parser.InstitutionCode() != "seabank" {
		t.Fatalf("expected institution code seabank, got %s", parser.InstitutionCode())
	}

	csvData := `Waktu Transaksi,Jenis Transaksi,Nominal,Status,No. Transaksi
2026-10-12 14:30:00,Transfer Keluar,150000,Berhasil,SB123456789
2026-10-13 09:00:00,Bunga Tabungan,15000,Berhasil,SB987654321
`
	if !parser.CanParse("seabank.csv", []byte(csvData[:40])) {
		t.Fatalf("expected CanParse true for SeaBank CSV")
	}

	records, err := parser.Parse(context.Background(), strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("failed to parse SeaBank CSV: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Amount != 150000 || records[0].Type != domain.TransactionExpense {
		t.Errorf("record 0 mismatch: %v", records[0])
	}
	if records[1].Amount != 15000 || records[1].Type != domain.TransactionIncome {
		t.Errorf("record 1 mismatch: %v", records[1])
	}
}

func TestGoPayParser(t *testing.T) {
	parser := adapters.NewGoPayParser()
	if parser.InstitutionCode() != "gopay" {
		t.Fatalf("expected institution code gopay, got %s", parser.InstitutionCode())
	}

	csvData := `Transaction Date,Transaction ID,Type,Description,Amount (IDR),Status
2026-10-14 12:00:00,GP111,Payment,GoFood Ayam Geprek,-35000,SUCCESS
2026-10-14 15:00:00,GP222,Top Up,Top Up from BCA,50000,SUCCESS
`
	if !parser.CanParse("gopay.csv", []byte(csvData[:40])) {
		t.Fatalf("expected CanParse true for GoPay CSV")
	}

	records, err := parser.Parse(context.Background(), strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("failed to parse GoPay CSV: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Amount != 35000 || records[0].Type != domain.TransactionExpense {
		t.Errorf("record 0 mismatch: %v", records[0])
	}
	if records[1].Amount != 50000 || records[1].Type != domain.TransactionIncome {
		t.Errorf("record 1 mismatch: %v", records[1])
	}
}

func TestDANAParser(t *testing.T) {
	parser := adapters.NewDANAParser()
	if parser.InstitutionCode() != "dana" {
		t.Fatalf("expected institution code dana, got %s", parser.InstitutionCode())
	}

	csvData := `Date,Description,Transaction Type,Amount,Balance
2026-10-15 11:20:00,DANA QRIS Indomaret,Expense,42000,108000
2026-10-15 16:00:00,Transfer Bank Masuk,Income,100000,208000
`
	if !parser.CanParse("dana.csv", []byte(csvData[:40])) {
		t.Fatalf("expected CanParse true for DANA CSV")
	}

	records, err := parser.Parse(context.Background(), strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("failed to parse DANA CSV: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Amount != 42000 || records[0].Type != domain.TransactionExpense {
		t.Errorf("record 0 mismatch: %v", records[0])
	}
	if records[1].Amount != 100000 || records[1].Type != domain.TransactionIncome {
		t.Errorf("record 1 mismatch: %v", records[1])
	}
}

func TestShopeePayParser(t *testing.T) {
	parser := adapters.NewShopeePayParser()
	if parser.InstitutionCode() != "shopeepay" {
		t.Fatalf("expected institution code shopeepay, got %s", parser.InstitutionCode())
	}

	csvData := `Date,Reference ID,Description,Amount,Status
2026-10-16 18:30:00,SP001,Pembayaran Pesanan Shopee,-89000,Completed
2026-10-16 20:00:00,SP002,Top up ShopeePay,100000,Completed
`
	if !parser.CanParse("shopeepay.csv", []byte(csvData[:40])) {
		t.Fatalf("expected CanParse true for ShopeePay CSV")
	}

	records, err := parser.Parse(context.Background(), strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("failed to parse ShopeePay CSV: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Amount != 89000 || records[0].Type != domain.TransactionExpense {
		t.Errorf("record 0 mismatch: %v", records[0])
	}
	if records[1].Amount != 100000 || records[1].Type != domain.TransactionIncome {
		t.Errorf("record 1 mismatch: %v", records[1])
	}
}

func TestGenericCSVParser(t *testing.T) {
	parser := adapters.NewGenericCSVParser()
	if parser.InstitutionCode() != "generic" {
		t.Fatalf("expected institution code generic, got %s", parser.InstitutionCode())
	}


	csvData := `Date,Description,Amount,Type
2026-10-17,Warung Makan,25000,expense
2026-10-18,Freelance Income,500000,income
`
	if !parser.CanParse("any_bank.csv", []byte(csvData[:40])) {
		t.Fatalf("expected CanParse true for Generic CSV")
	}

	records, err := parser.Parse(context.Background(), strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("failed to parse Generic CSV: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Amount != 25000 || records[0].Type != domain.TransactionExpense {
		t.Errorf("record 0 mismatch: %v", records[0])
	}
	if records[1].Amount != 500000 || records[1].Type != domain.TransactionIncome {
		t.Errorf("record 1 mismatch: %v", records[1])
	}
}
func TestDefaultRegistry(t *testing.T) {
	reg := adapters.NewDefaultRegistry()
	expectedCodes := []string{"bca", "mandiri", "jago", "seabank", "gopay", "dana", "shopeepay", "generic"}
	for _, code := range expectedCodes {
		if _, ok := reg.Get(code); !ok {
			t.Errorf("expected parser for %q to be registered in default registry", code)
		}
	}
}
