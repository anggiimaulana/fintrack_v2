package ingestion

import (
	"context"
	"regexp"
	"strings"
)

// NormalizedResult contains parsed metadata from raw transaction text.
type NormalizedResult struct {
	CleanMerchant      string  `json:"clean_merchant"`
	SuggestedCategory  string  `json:"suggested_category,omitempty"`
	Confidence         float64 `json:"confidence"`
	IsTransferCandidate bool    `json:"is_transfer_candidate"`
}

// AIFallbackNormalizer is invoked when Stage 1 regex confidence is low.
type AIFallbackNormalizer interface {
	NormalizeAI(ctx context.Context, rawText string) (*NormalizedResult, error)
}

// Normalizer handles Two-Stage description sanitization.
type Normalizer struct {
	aiFallback AIFallbackNormalizer

	reNoise        *regexp.Regexp
	reQRIS         *regexp.Regexp
	reTransfer     *regexp.Regexp
	reWhitespace   *regexp.Regexp
}

// NewNormalizer creates a regex-based Stage 1 normalizer with optional AI fallback.
func NewNormalizer(aiFallback ...AIFallbackNormalizer) *Normalizer {
	var fallback AIFallbackNormalizer
	if len(aiFallback) > 0 {
		fallback = aiFallback[0]
	}

	return &Normalizer{
		aiFallback: fallback,
		// Strip common machine prefixes: e.g. TRSF E-BANKING DB 2010/WS95011, M-BCA 20/10 WSID:88123, BI-FAST
		reNoise: regexp.MustCompile(`(?i)\b(TRSF|E-BANKING|DB|CR|BI-FAST|M-BCA|IBANK|SWITCHING|WSID:\w+|WS\d+|\d{2}/\d{2}|\d{4}/\w+)\b`),
		// Extract merchant after QRIS: e.g. QRIS SHOPEEPAY
		reQRIS: regexp.MustCompile(`(?i)QRIS\s+([A-Za-z0-9\s\.\-_]+)`),
		// Extract transfer partner: e.g. TRF KE KANTONG JAGO, TRF DR 014 ANGGA TO TOKOPEDIA
		reTransfer:   regexp.MustCompile(`(?i)\b(?:TRF|TRANSFER)\s+(?:TO|DR|KE|DARI)\s+(?:\d+\s+)*(?:(?:TO|KE)\s+)?([A-Za-z0-9\s\.\-_]+)`),
		reWhitespace: regexp.MustCompile(`\s+`),
	}
}

// NormalizeStage1 applies deterministic regex rules in-memory.
func (n *Normalizer) NormalizeStage1(raw string) *NormalizedResult {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return &NormalizedResult{CleanMerchant: "Transaksi", Confidence: 0.0}
	}

	// 1. Check QRIS pattern
	if match := n.reQRIS.FindStringSubmatch(clean); len(match) > 1 {
		merchant := n.cleanEntity(match[1])
		if merchant != "" {
			return &NormalizedResult{
				CleanMerchant: merchant,
				Confidence:    0.95,
			}
		}
	}

	// 2. Check Transfer pattern
	if match := n.reTransfer.FindStringSubmatch(clean); len(match) > 1 {
		target := match[1]
		upperTarget := strings.ToUpper(target)
		if idx := strings.LastIndex(upperTarget, " TO "); idx != -1 {
			target = target[idx+4:]
		} else if idx := strings.LastIndex(upperTarget, " KE "); idx != -1 {
			target = target[idx+4:]
		}

		merchant := n.cleanEntity(target)
		if merchant != "" {
			return &NormalizedResult{
				CleanMerchant:       merchant,
				Confidence:          0.90,
				IsTransferCandidate: true,
			}
		}
	}

	// 3. Fallback: Strip noise tokens
	stripped := n.reNoise.ReplaceAllString(clean, " ")
	stripped = n.reWhitespace.ReplaceAllString(stripped, " ")
	stripped = n.cleanEntity(stripped)

	if stripped == "" {
		stripped = "Transaksi"
	}

	confidence := 0.75
	if stripped == "Transaksi" {
		confidence = 0.20
	}

	return &NormalizedResult{
		CleanMerchant: stripped,
		Confidence:    confidence,
	}
}

// Normalize performs Stage 1, falling back to Stage 2 AI Gateway if confidence is low.
func (n *Normalizer) Normalize(ctx context.Context, raw string) *NormalizedResult {
	res := n.NormalizeStage1(raw)
	if res.Confidence >= 0.70 || n.aiFallback == nil {
		return res
	}

	aiRes, err := n.aiFallback.NormalizeAI(ctx, raw)
	if err == nil && aiRes != nil {
		return aiRes
	}
	return res
}

func (n *Normalizer) cleanEntity(s string) string {
	s = strings.TrimSpace(s)
	// Remove common leading noise words
	tokens := strings.Fields(s)
	filtered := make([]string, 0, len(tokens))
	for _, t := range tokens {
		upper := strings.ToUpper(t)
		if upper == "TO" || upper == "KE" || upper == "DR" || upper == "DARI" || upper == "VIA" || upper == "DENGAN" {
			continue
		}
		filtered = append(filtered, toTitleWord(t))
	}
	return strings.Join(filtered, " ")
}

func toTitleWord(w string) string {
	lower := strings.ToLower(w)
	if len(lower) == 0 {
		return ""
	}
	// Keep brand capitalization for well-known acronyms
	switch lower {
	case "qris":
		return "QRIS"
	case "bca":
		return "BCA"
	case "bni":
		return "BNI"
	case "bri":
		return "BRI"
	case "pln":
		return "PLN"
	case "shopeepay":
		return "ShopeePay"
	case "gopay":
		return "GoPay"
	}
	return strings.ToUpper(lower[:1]) + lower[1:]
}
