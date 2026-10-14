package export

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/fintrack/backend/internal/domain"
)

func sampleInput() ReportInput {
	return ReportInput{
		Report: &domain.ReportResponse{
			Summary: domain.ReportSummary{TotalIncome: 5_000_000, TotalExpense: 3_200_000, TotalSavings: 800_000, Surplus: 1_800_000},
			Categories: []domain.CategoryReport{
				{CategoryName: "Makanan & Minuman", Total: 1_200_000, Percentage: 37.5, Trend: 12.3, Count: 24},
				{CategoryName: "Transportasi", Total: 600_000, Percentage: 18.75, Trend: -5.1, Count: 10},
			},
			Monthly: []domain.MonthlyComparison{
				{Month: "Sep 2026", Income: 4_800_000, Expense: 3_000_000, Savings: 700_000},
				{Month: "Okt 2026", Income: 5_000_000, Expense: 3_200_000, Savings: 800_000},
			},
		},
		Type:        "expense",
		StartDate:   "2026-10-01",
		EndDate:     "2026-10-31",
		Period:      "2026-10",
		GeneratedAt: time.Date(2026, 10, 14, 10, 30, 0, 0, time.UTC),
	}
}

func TestParseFormat(t *testing.T) {
	cases := []struct {
		in      string
		want    Format
		wantOK  bool
	}{
		{"csv", FormatCSV, true},
		{"xlsx", FormatXLSX, true},
		{"pdf", FormatPDF, true},
		{"", FormatCSV, true},
		{"docx", "", false},
		{"CSV", "", false},
	}
	for _, tc := range cases {
		got, ok := ParseFormat(tc.in)
		if ok != tc.wantOK || got != tc.want {
			t.Errorf("ParseFormat(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}

func TestFilename(t *testing.T) {
	if got := FormatCSV.Filename("2026-10"); got != "laporan-2026-10.csv" {
		t.Errorf("csv filename = %q", got)
	}
	if got := FormatPDF.Filename(""); !strings.HasSuffix(got, ".pdf") {
		t.Errorf("pdf filename = %q", got)
	}
}

func TestContentType(t *testing.T) {
	if FormatXLSX.ContentType() != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Error("xlsx content type wrong")
	}
	if FormatPDF.ContentType() != "application/pdf" {
		t.Error("pdf content type wrong")
	}
	if !strings.HasPrefix(FormatCSV.ContentType(), "text/csv") {
		t.Error("csv content type wrong")
	}
}

func TestRenderCSV(t *testing.T) {
	out, err := Render(sampleInput(), FormatCSV)
	if err != nil {
		t.Fatalf("render csv: %v", err)
	}
	// UTF-8 BOM so Excel opens it correctly.
	if !bytes.HasPrefix(out, []byte{0xEF, 0xBB, 0xBF}) {
		t.Error("csv missing UTF-8 BOM")
	}
	s := string(out)
	for _, want := range []string{
		"Laporan Keuangan FinTrack",
		"Pengeluaran",
		"Makanan & Minuman",
		"Transportasi",
		"Perbandingan Bulanan",
		"Sep 2026",
		"Rp 5000000",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("csv missing %q", want)
		}
	}
}

func TestRenderCSV_EscapesCommas(t *testing.T) {
	in := sampleInput()
	in.Report.Categories[0].CategoryName = "Makan, Minum & Jajan"
	out, err := Render(in, FormatCSV)
	if err != nil {
		t.Fatalf("render csv: %v", err)
	}
	// encoding/csv quotes a field containing a comma.
	if !strings.Contains(string(out), `"Makan, Minum & Jajan"`) {
		t.Error("csv did not quote a field containing a comma")
	}
}

func TestRenderXLSX_IsAValidWorkbook(t *testing.T) {
	out, err := Render(sampleInput(), FormatXLSX)
	if err != nil {
		t.Fatalf("render xlsx: %v", err)
	}
	// Re-open the bytes: proves the OOXML package is well-formed, not just non-empty.
	f, err := excelize.OpenReader(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("xlsx not readable: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Laporan")
	if err != nil {
		t.Fatalf("xlsx missing sheet 'Laporan': %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("xlsx sheet is empty")
	}
	// The title must be present in the first row.
	if len(rows[0]) == 0 || rows[0][0] != "Laporan Keuangan FinTrack" {
		t.Errorf("xlsx title row = %v", rows[0])
	}
	// And a category must appear somewhere.
	found := false
	for _, r := range rows {
		if len(r) > 0 && r[0] == "Makanan & Minuman" {
			found = true
			break
		}
	}
	if !found {
		t.Error("xlsx missing category row")
	}
}

func TestRenderPDF_IsStructurallyValid(t *testing.T) {
	out, err := Render(sampleInput(), FormatPDF)
	if err != nil {
		t.Fatalf("render pdf: %v", err)
	}
	s := string(out)
	if !strings.HasPrefix(s, "%PDF-1.") {
		t.Errorf("pdf missing header, got %q", s[:min(8, len(s))])
	}
	if !strings.HasSuffix(s, "%%EOF\n") {
		t.Error("pdf missing EOF marker")
	}
	if !strings.Contains(s, "startxref") || !strings.Contains(s, "trailer") {
		t.Error("pdf missing trailer/startxref")
	}
	// The xref offset must point at the "xref" keyword.
	idx := strings.Index(s, "startxref\n")
	if idx < 0 {
		t.Fatal("no startxref")
	}
	offStr := strings.TrimSpace(strings.SplitN(s[idx+len("startxref\n"):], "\n", 2)[0])
	off := 0
	for _, ch := range offStr {
		if ch < '0' || ch > '9' {
			t.Fatalf("startxref not numeric: %q", offStr)
		}
		off = off*10 + int(ch-'0')
	}
	if off <= 0 || off >= len(s) {
		t.Fatalf("startxref offset %d out of range (len %d)", off, len(s))
	}
	if !strings.HasPrefix(s[off:], "xref") {
		t.Errorf("startxref offset %d does not point at 'xref'", off)
	}
	// The category name (ASCII) should be present in a content stream.
	if !strings.Contains(s, "Makanan & Minuman") {
		t.Error("pdf content missing category name")
	}
}

func TestRenderPDF_Paginates(t *testing.T) {
	in := sampleInput()
	// Enough categories to overflow one page.
	for i := range 120 {
		in.Report.Categories = append(in.Report.Categories, domain.CategoryReport{
			CategoryName: "Kategori " + strings.Repeat("X", 3) + string(rune('A'+i%26)),
			Total:        float64(i) * 1000,
			Percentage:   1.5,
			Count:        i,
		})
	}
	out, err := Render(in, FormatPDF)
	if err != nil {
		t.Fatalf("render pdf: %v", err)
	}
	s := string(out)
	// A multi-page document declares /Count > 1 and has multiple page objects.
	if !strings.Contains(s, "/Type /Pages") {
		t.Fatal("missing Pages object")
	}
	if strings.Count(s, "/Type /Page ") < 2 {
		t.Errorf("expected multiple page objects, found %d", strings.Count(s, "/Type /Page "))
	}
}

func TestRenderPDF_EscapesParentheses(t *testing.T) {
	in := sampleInput()
	in.Report.Categories[0].CategoryName = "Makanan (luar)"
	out, err := Render(in, FormatPDF)
	if err != nil {
		t.Fatalf("render pdf: %v", err)
	}
	// Unescaped parens would break the content stream string literal.
	if !strings.Contains(string(out), `Makanan \(luar\)`) {
		t.Error("pdf did not escape parentheses in a string literal")
	}
}

func TestRender_UnknownFormatErrors(t *testing.T) {
	if _, err := Render(sampleInput(), Format("docx")); err == nil {
		t.Fatal("expected an error for an unsupported format")
	}
}
