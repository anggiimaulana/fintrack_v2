package adapters

import (
	"encoding/csv"
	"io"
	"math"
	"strconv"
	"strings"
	"time"
)

// ParseIndonesianAmount parses amount strings supporting Indonesian comma decimal (50.000,00)
// and standard dot decimal (50,000.00 or 50000.00 or -35000).
func ParseIndonesianAmount(s string) (float64, bool) {
	clean := strings.TrimSpace(s)
	clean = strings.ReplaceAll(clean, "Rp", "")
	clean = strings.ReplaceAll(clean, "IDR", "")
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return 0, false
	}

	isNegative := false
	if strings.HasPrefix(clean, "-") {
		isNegative = true
		clean = strings.TrimPrefix(clean, "-")
	} else if strings.HasPrefix(clean, "(") && strings.HasSuffix(clean, ")") {
		isNegative = true
		clean = strings.TrimPrefix(strings.TrimSuffix(clean, ")"), "(")
	}
	clean = strings.TrimSpace(clean)

	// Determine if comma is decimal separator: e.g. 50.000,00
	if strings.Contains(clean, ",") && strings.Contains(clean, ".") {
		lastComma := strings.LastIndex(clean, ",")
		lastDot := strings.LastIndex(clean, ".")
		if lastComma > lastDot {
			// e.g. 1.234.567,89
			clean = strings.ReplaceAll(clean, ".", "")
			clean = strings.ReplaceAll(clean, ",", ".")
		} else {
			// e.g. 1,234,567.89
			clean = strings.ReplaceAll(clean, ",", "")
		}
	} else if strings.Contains(clean, ",") {
		// e.g. 50000,00 -> decimal
		// or 50,000 -> thousands? If exactly 2 digits after comma, it's decimal
		parts := strings.Split(clean, ",")
		if len(parts) == 2 && len(parts[1]) <= 2 {
			clean = strings.ReplaceAll(clean, ",", ".")
		} else {
			clean = strings.ReplaceAll(clean, ",", "")
		}
	}

	val, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0, false
	}
	if isNegative {
		val = -math.Abs(val)
	}
	return val, true
}

// ParseFlexibleDate tries common Indonesian and standard ISO date formats.
func ParseFlexibleDate(s string) (time.Time, bool) {
	clean := strings.TrimSpace(s)
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02",
		"02/01/2006 15:04:05",
		"02/01/2006",
		"02-01-2006 15:04:05",
		"02-01-2006",
		"2006/01/02",
	}

	for _, f := range formats {
		if t, err := time.Parse(f, clean); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// ReadCSVRecords reads all records with flexible comma or semicolon delimiter.
func ReadCSVRecords(r io.Reader) ([][]string, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(strings.NewReader(string(content)))
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1

	// Check if semicolon delimited
	if strings.Count(string(content), ";") > strings.Count(string(content), ",") {
		reader.Comma = ';'
	}

	return reader.ReadAll()
}
