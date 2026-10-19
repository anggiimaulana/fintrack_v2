// Package export renders a financial report into CSV, XLSX, or PDF bytes.
//
// It is deliberately transport-free (no HTTP, no Gin) and depends only on the
// report domain, so it is unit-testable without a server or database. The HTTP
// handler decides the format and streams the result.
package export

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/fintrack/backend/internal/domain"
)

// Format is a supported export format.
type Format string

const (
	FormatCSV  Format = "csv"
	FormatXLSX Format = "xlsx"
	FormatPDF  Format = "pdf"
)

// ParseFormat validates and normalises a requested format, defaulting to CSV.
func ParseFormat(s string) (Format, bool) {
	switch Format(s) {
	case FormatCSV:
		return FormatCSV, true
	case FormatXLSX:
		return FormatXLSX, true
	case FormatPDF:
		return FormatPDF, true
	case "":
		return FormatCSV, true
	default:
		return "", false
	}
}

// ContentType returns the MIME type for a format.
func (f Format) ContentType() string {
	switch f {
	case FormatXLSX:
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case FormatPDF:
		return "application/pdf"
	default:
		return "text/csv; charset=utf-8"
	}
}

// Filename builds a download filename, e.g. "laporan-2026-10.csv".
func (f Format) Filename(period string) string {
	if period == "" {
		period = time.Now().Format("2006-01-02")
	}
	return fmt.Sprintf("laporan-%s.%s", period, f)
}

// ReportInput carries the report plus the metadata the export header needs.
type ReportInput struct {
	Report    *domain.ReportResponse
	Type      string // expense | income | savings
	StartDate string
	EndDate   string
	Period    string
	// CurrencySymbol/CurrencyDigits describe how to render the report currency.
	// Optional: when empty/nil the built-in defaults are used, so the export
	// package stays usable without a catalog. Populated from the dynamic catalog
	// so a currency added by a Super Admin renders with its configured appearance.
	CurrencySymbol string
	CurrencyDigits *int
	// GeneratedAt is injectable so tests are deterministic.
	GeneratedAt time.Time
}

// typeLabel renders the report type in Indonesian for the document header.
func typeLabel(t string) string {
	switch t {
	case "income":
		return "Pemasukan"
	case "savings":
		return "Tabungan"
	default:
		return "Pengeluaran"
	}
}

// Render produces the bytes for the requested format.
func Render(in ReportInput, format Format) ([]byte, error) {
	switch format {
	case FormatCSV:
		return renderCSV(in)
	case FormatXLSX:
		return renderXLSX(in)
	case FormatPDF:
		return renderPDF(in)
	default:
		return nil, fmt.Errorf("format tidak didukung: %s", format)
	}
}

// ─── CSV ──────────────────────────────────────────────────────

func renderCSV(in ReportInput) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	writeRow := func(cells ...string) error { return w.Write(cells) }

	if err := writeRow("Laporan Keuangan FinTrack"); err != nil {
		return nil, err
	}
	code := reportCurrency(in)
	if err := writeRow("Jenis", typeLabel(in.Type)); err != nil {
		return nil, err
	}
	if err := writeRow("Periode", in.StartDate, in.EndDate); err != nil {
		return nil, err
	}
	if err := writeRow("Currency", code); err != nil {
		return nil, err
	}
	if legend := rateLegend(in); legend != "" {
		if err := writeRow("Rate", legend); err != nil {
			return nil, err
		}
	}
	if err := writeRow("Dibuat", in.GeneratedAt.Format("2006-01-02 15:04")); err != nil {
		return nil, err
	}
	if err := writeRow(""); err != nil {
		return nil, err
	}

	// Summary
	if err := writeRow("Ringkasan"); err != nil {
		return nil, err
	}
	if err := writeRow("Total Pemasukan", in.money(in.Report.Summary.TotalIncome)); err != nil {
		return nil, err
	}
	if err := writeRow("Total Pengeluaran", in.money(in.Report.Summary.TotalExpense)); err != nil {
		return nil, err
	}
	if err := writeRow("Total Tabungan", in.money(in.Report.Summary.TotalSavings)); err != nil {
		return nil, err
	}
	if err := writeRow("Surplus", in.money(in.Report.Summary.Surplus)); err != nil {
		return nil, err
	}
	if err := writeRow(""); err != nil {
		return nil, err
	}

	// Categories
	if err := writeRow("Detail per Kategori"); err != nil {
		return nil, err
	}
	if err := writeRow("Kategori", "Total", "Persentase", "Trend (%)", "Jumlah Transaksi"); err != nil {
		return nil, err
	}
	for _, c := range in.Report.Categories {
		if err := writeRow(c.CategoryName, in.money(c.Total), pct(c.Percentage), trend(c.Trend), strconv.Itoa(c.Count)); err != nil {
			return nil, err
		}
	}
	if err := writeRow(""); err != nil {
		return nil, err
	}

	// Monthly
	if err := writeRow("Perbandingan Bulanan"); err != nil {
		return nil, err
	}
	if err := writeRow("Periode", "Pemasukan", "Pengeluaran", "Tabungan"); err != nil {
		return nil, err
	}
	for _, m := range in.Report.Monthly {
		if err := writeRow(m.Month, in.money(m.Income), in.money(m.Expense), in.money(m.Savings)); err != nil {
			return nil, err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	// Excel opens UTF-8 CSV correctly only with a BOM.
	return append([]byte{0xEF, 0xBB, 0xBF}, buf.Bytes()...), nil
}

// ─── XLSX ─────────────────────────────────────────────────────

func renderXLSX(in ReportInput) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	const sheet = "Laporan"
	f.SetSheetName("Sheet1", sheet)

	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 12}})
	if err != nil {
		return nil, err
	}
	header, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"1F2937"}},
	})
	if err != nil {
		return nil, err
	}

	// Title block
	_ = f.SetCellValue(sheet, "A1", "Laporan Keuangan FinTrack")
	_ = f.SetCellStyle(sheet, "A1", "A1", bold)
	_ = f.SetCellValue(sheet, "A2", "Jenis")
	_ = f.SetCellValue(sheet, "B2", typeLabel(in.Type))
	_ = f.SetCellValue(sheet, "A3", "Periode")
	_ = f.SetCellValue(sheet, "B3", fmt.Sprintf("%s s/d %s", in.StartDate, in.EndDate))
	_ = f.SetCellValue(sheet, "A4", "Dibuat")
	_ = f.SetCellValue(sheet, "B4", in.GeneratedAt.Format("2006-01-02 15:04"))
	_ = f.SetCellValue(sheet, "A5", "Currency")
	currLabel := reportCurrency(in)
	if legend := rateLegend(in); legend != "" {
		currLabel = currLabel + " (" + legend + ")"
	}
	_ = f.SetCellValue(sheet, "B5", currLabel)

	// Summary
	_ = f.SetCellValue(sheet, "A6", "Ringkasan")
	_ = f.SetCellStyle(sheet, "A6", "A6", bold)
	summary := [][2]any{
		{"Total Pemasukan", in.Report.Summary.TotalIncome},
		{"Total Pengeluaran", in.Report.Summary.TotalExpense},
		{"Total Tabungan", in.Report.Summary.TotalSavings},
		{"Surplus", in.Report.Summary.Surplus},
	}
	for i, row := range summary {
		cell := fmt.Sprintf("A%d", 7+i)
		_ = f.SetCellValue(sheet, cell, row[0])
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", 7+i), row[1])
	}

	// Categories
	catHeaderRow := 12
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", catHeaderRow), "Detail per Kategori")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", catHeaderRow), fmt.Sprintf("A%d", catHeaderRow), bold)
	catCols := []string{"Kategori", "Total", "Persentase (%)", "Trend (%)", "Jumlah"}
	for i, h := range catCols {
		cell, _ := excelize.CoordinatesToCellName(i+1, catHeaderRow+1)
		_ = f.SetCellValue(sheet, cell, h)
		_ = f.SetCellStyle(sheet, cell, cell, header)
	}
	for r, c := range in.Report.Categories {
		row := catHeaderRow + 2 + r
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), c.CategoryName)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), c.Total)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", row), c.Percentage)
		_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", row), c.Trend)
		_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), c.Count)
	}

	// Monthly
	monHeaderRow := catHeaderRow + 2 + len(in.Report.Categories) + 1
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", monHeaderRow), "Perbandingan Bulanan")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", monHeaderRow), fmt.Sprintf("A%d", monHeaderRow), bold)
	monCols := []string{"Periode", "Pemasukan", "Pengeluaran", "Tabungan"}
	for i, h := range monCols {
		cell, _ := excelize.CoordinatesToCellName(i+1, monHeaderRow+1)
		_ = f.SetCellValue(sheet, cell, h)
		_ = f.SetCellStyle(sheet, cell, cell, header)
	}
	for r, m := range in.Report.Monthly {
		row := monHeaderRow + 2 + r
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), m.Month)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), m.Income)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", row), m.Expense)
		_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", row), m.Savings)
	}

	_ = f.SetColWidth(sheet, "A", "A", 28)
	_ = f.SetColWidth(sheet, "B", "E", 18)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ─── helpers ──────────────────────────────────────────────────

// reportCurrency returns the currency the report totals are expressed in.
func reportCurrency(in ReportInput) string {
	if in.Report != nil && in.Report.CurrencyCode != "" {
		return in.Report.CurrencyCode
	}
	return "IDR"
}

// money formats a report amount in the report's currency. The report endpoint
// already converted the aggregates to the display currency (at the period-end
// rate), so no further conversion happens here.
func (in ReportInput) money(v float64) string {
	code := reportCurrency(in)
	sym := in.CurrencySymbol
	if sym == "" {
		sym = currencySymbol(code)
	}
	digits := currencyDigits(code)
	if in.CurrencyDigits != nil {
		digits = *in.CurrencyDigits
	}
	return sym + " " + strconv.FormatFloat(v, 'f', digits, 64)
}

// currencySymbol maps a code to its display symbol. Unknown codes fall back to
// the code itself so a newly added currency still renders legibly.
func currencySymbol(code string) string {
	switch code {
	case "IDR":
		return "Rp"
	case "JPY":
		return "¥"
	case "USD":
		return "$"
	case "MYR":
		return "RM"
	default:
		return code + " "
	}
}

// currencyDigits returns the minor-unit precision for known currencies.
func currencyDigits(code string) int {
	switch code {
	case "IDR", "JPY":
		return 0
	default:
		return 2
	}
}

// money formats an amount in the given currency using its symbol and precision.
func money(v float64, code string) string {
	return currencySymbol(code) + " " + strconv.FormatFloat(v, 'f', currencyDigits(code), 64)
}

// rupiah is kept for callers that always render base currency.
func rupiah(v float64) string { return money(v, "IDR") }

// rateLegend describes the conversion used when a report is not in base IDR.
func rateLegend(in ReportInput) string {
	code := reportCurrency(in)
	if code == "IDR" || in.Report == nil || in.Report.RateUsed == 0 {
		return ""
	}
	at := ""
	if in.Report.RateAt != nil {
		at = in.Report.RateAt.Format("2006-01-02")
	}
	return fmt.Sprintf("1 %s = %.4f IDR (rate as of %s)", code, in.Report.RateUsed, at)
}

func pct(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

func trend(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}
