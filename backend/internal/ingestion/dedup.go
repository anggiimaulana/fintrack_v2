package ingestion

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// GenerateFingerprint creates a deterministic SHA-256 fingerprint for a transaction row.
// Normalizes whitespace, date to YYYY-MM-DD, amount to 2 decimal places, and casing.
func GenerateFingerprint(userID, accountID uuid.UUID, date time.Time, amount float64, rawDesc string) string {
	normalizedDate := date.Format("2006-01-02")
	normalizedAmount := fmt.Sprintf("%.2f", amount)
	cleaned := strings.ToLower(strings.Join(strings.Fields(rawDesc), " "))

	h := sha256.New()
	h.Write([]byte(userID.String() + "|" + accountID.String() + "|" + normalizedDate + "|" + normalizedAmount + "|" + cleaned))
	return hex.EncodeToString(h.Sum(nil))
}
