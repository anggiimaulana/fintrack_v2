package export

import (
	"bytes"
	"fmt"
	"strings"
)

// A deliberately small, dependency-free PDF writer.
//
// The project has no PDF library and adding one (gofpdf/fpdf) for a single
// report would be a heavy, security-reviewed dependency. A report is a simple
// paginated table of text, so a minimal PDF 1.4 writer over the standard
// Helvetica font covers it: no embedded fonts, no compression, ~150 lines.
//
// Coordinates are in PDF points (72/inch) with the origin at the bottom-left.

const (
	pageWidth  = 595.28 // A4
	pageHeight = 841.89
	marginX    = 48.0
	marginTop  = 60.0
	marginBottom = 60.0
	lineHeight = 16.0
	fontSize   = 10.0
	titleSize  = 16.0
)

// pdfDoc accumulates content-stream operators for one or more pages.
type pdfDoc struct {
	pages []string
	buf   bytes.Buffer
	y     float64
}

func newPDFDoc() *pdfDoc {
	return &pdfDoc{y: pageHeight - marginTop}
}

// newPage starts a page and writes the running header.
func (d *pdfDoc) newPage() {
	if d.buf.Len() > 0 {
		d.pages = append(d.pages, d.buf.String())
		d.buf.Reset()
	}
	d.y = pageHeight - marginTop
}

// ensureSpace breaks to a new page when the cursor would run past the bottom.
func (d *pdfDoc) ensureSpace(needed float64) {
	if d.y-needed < marginBottom {
		d.newPage()
	}
}

// text writes a single line at the current cursor and advances it.
func (d *pdfDoc) text(s string, size float64, bold bool) {
	d.ensureSpace(lineHeight)
	font := "F1"
	if bold {
		font = "F2"
	}
	fmt.Fprintf(&d.buf, "BT /%s %.2f Tf %.2f %.2f Td (%s) Tj ET\n", font, size, marginX, d.y, escapePDF(s))
	d.y -= lineHeight
}

// textAt writes a line at an explicit x offset (for columns).
func (d *pdfDoc) textAt(x, y float64, s string, size float64, bold bool) {
	font := "F1"
	if bold {
		font = "F2"
	}
	fmt.Fprintf(&d.buf, "BT /%s %.2f Tf %.2f %.2f Td (%s) Tj ET\n", font, size, x, y, escapePDF(s))
}

// hline draws a horizontal rule at the current cursor.
func (d *pdfDoc) hline() {
	d.ensureSpace(4)
	fmt.Fprintf(&d.buf, "0.6 w %.2f %.2f m %.2f %.2f l S\n", marginX, d.y, pageWidth-marginX, d.y)
	d.y -= 8
}

func (d *pdfDoc) finish() []byte {
	if d.buf.Len() > 0 {
		d.pages = append(d.pages, d.buf.String())
		d.buf.Reset()
	}
	return d.assemble()
}

// escapePDF escapes the characters that are special inside a PDF string literal.
func escapePDF(s string) string {
	r := strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)")
	// The standard 14 fonts use WinAnsi; drop anything non-ASCII so we never
	// emit an invalid byte sequence for a missing glyph.
	ascii := strings.Map(func(c rune) rune {
		if c < 32 || c > 126 {
			return -1
		}
		return c
	}, s)
	return r.Replace(ascii)
}

// assemble writes the PDF object graph (catalog, pages, fonts, content streams)
// and the cross-reference table.
func (d *pdfDoc) assemble() []byte {
	var out bytes.Buffer
	// PDF header. The comment line with high bytes marks the file as binary so
	// tools never mangle it as text.
	out.WriteString("%PDF-1.4\n")
	out.WriteString("%\xE2\xE3\xCF\xD3\n")

	// offsets[i] is the byte offset of object i+1.
	offsets := make([]int, 0, len(d.pages)*2+4)
	writeObj := func(body string) {
		num := len(offsets) + 1
		offsets = append(offsets, out.Len())
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", num, body)
	}

	// Layout: 1 catalog, 2 pages, 3 font F1, 4 font F2, then per page: content + page.
	contentStart := 5
	pageObjNums := make([]int, len(d.pages))
	contentObjNums := make([]int, len(d.pages))
	for i := range d.pages {
		contentObjNums[i] = contentStart + i*2
		pageObjNums[i] = contentStart + i*2 + 1
	}

	// 1: Catalog
	writeObj("<< /Type /Catalog /Pages 2 0 R >>")
	// 2: Pages
	var kids strings.Builder
	for _, n := range pageObjNums {
		fmt.Fprintf(&kids, "%d 0 R ", n)
	}
	writeObj(fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.TrimSpace(kids.String()), len(d.pages)))
	// 3: Helvetica
	writeObj("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	// 4: Helvetica-Bold
	writeObj("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>")

	for i, content := range d.pages {
		writeObj(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content)+1, content))
		writeObj(fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Resources << /Font << /F1 3 0 R /F2 4 0 R >> >> /Contents %d 0 R >>",
			pageWidth, pageHeight, contentObjNums[i]))
	}

	// Cross-reference table.
	xrefOffset := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n", len(offsets)+1)
	out.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		fmt.Fprintf(&out, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets)+1, xrefOffset)

	return out.Bytes()
}

// renderPDF builds the report PDF.
func renderPDF(in ReportInput) ([]byte, error) {
	d := newPDFDoc()
	d.newPage()

	d.text("Laporan Keuangan FinTrack", titleSize, true)
	d.text(fmt.Sprintf("Jenis: %s", typeLabel(in.Type)), fontSize, false)
	d.text(fmt.Sprintf("Periode: %s s/d %s", in.StartDate, in.EndDate), fontSize, false)
	d.text(fmt.Sprintf("Dibuat: %s", in.GeneratedAt.Format("2006-01-02 15:04")), fontSize, false)
	code := reportCurrency(in)
	d.text(fmt.Sprintf("Currency: %s", code), fontSize, false)
	if legend := rateLegend(in); legend != "" {
		d.text(fmt.Sprintf("Rate: %s", legend), fontSize, false)
	}
	d.y -= 6

	d.text("Ringkasan", 12, true)
	d.text(fmt.Sprintf("Total Pemasukan   : %s", in.money(in.Report.Summary.TotalIncome)), fontSize, false)
	d.text(fmt.Sprintf("Total Pengeluaran : %s", in.money(in.Report.Summary.TotalExpense)), fontSize, false)
	d.text(fmt.Sprintf("Total Tabungan    : %s", in.money(in.Report.Summary.TotalSavings)), fontSize, false)
	d.text(fmt.Sprintf("Surplus           : %s", in.money(in.Report.Summary.Surplus)), fontSize, false)
	d.y -= 8

	d.text("Detail per Kategori", 12, true)
	colX := []float64{marginX, marginX + 200, marginX + 300, marginX + 380, marginX + 460}
	d.ensureSpace(lineHeight)
	headerY := d.y
	d.textAt(colX[0], headerY, "Kategori", fontSize, true)
	d.textAt(colX[1], headerY, "Total", fontSize, true)
	d.textAt(colX[2], headerY, "%", fontSize, true)
	d.textAt(colX[3], headerY, "Trend", fontSize, true)
	d.textAt(colX[4], headerY, "Jml", fontSize, true)
	d.y -= lineHeight
	d.hline()

	if len(in.Report.Categories) == 0 {
		d.text("Belum ada data kategori untuk periode ini.", fontSize, false)
	}
	for _, c := range in.Report.Categories {
		d.ensureSpace(lineHeight)
		y := d.y
		d.textAt(colX[0], y, truncate(c.CategoryName, 30), fontSize, false)
		d.textAt(colX[1], y, in.money(c.Total), fontSize, false)
		d.textAt(colX[2], y, pct(c.Percentage), fontSize, false)
		d.textAt(colX[3], y, trend(c.Trend), fontSize, false)
		d.textAt(colX[4], y, fmt.Sprintf("%d", c.Count), fontSize, false)
		d.y -= lineHeight
	}
	d.y -= 8

	d.text("Perbandingan Bulanan", 12, true)
	mColX := []float64{marginX, marginX + 200, marginX + 300, marginX + 400}
	d.ensureSpace(lineHeight)
	headerY = d.y
	d.textAt(mColX[0], headerY, "Periode", fontSize, true)
	d.textAt(mColX[1], headerY, "Pemasukan", fontSize, true)
	d.textAt(mColX[2], headerY, "Pengeluaran", fontSize, true)
	d.textAt(mColX[3], headerY, "Tabungan", fontSize, true)
	d.y -= lineHeight
	d.hline()

	if len(in.Report.Monthly) == 0 {
		d.text("Belum ada data bulanan untuk periode ini.", fontSize, false)
	}
	for _, m := range in.Report.Monthly {
		d.ensureSpace(lineHeight)
		y := d.y
		d.textAt(mColX[0], y, m.Month, fontSize, false)
		d.textAt(mColX[1], y, in.money(m.Income), fontSize, false)
		d.textAt(mColX[2], y, in.money(m.Expense), fontSize, false)
		d.textAt(mColX[3], y, in.money(m.Savings), fontSize, false)
		d.y -= lineHeight
	}

	return d.finish(), nil
}

// truncate shortens a label to max runes, appending an ellipsis marker.
func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "~"
}
