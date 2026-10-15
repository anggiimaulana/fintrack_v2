// Package usage exposes a user's own consumption: a time series for charts and
// a paginated history ledger. It reads from the existing append-only
// `token_ledger` (AI tokens/requests, OCR, voice) and the `transactions` table,
// so no new producer needs to write anything — the data is always consistent
// with what the quota gates enforce.
package usage

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Granularity is the bucket size of a time series.
type Granularity string

const (
	GranularityHour  Granularity = "hour"
	GranularityDay   Granularity = "day"
	GranularityMonth Granularity = "month"
)

// Event kinds, one per consumption source.
const (
	KindTransaction = "transaction"
	KindAIToken     = "ai_token"
	KindAIRequest   = "ai_request"
	KindOCR         = "ocr"
	KindVoice       = "voice"
)

// Window is a resolved time range plus the bucket size to aggregate it with.
type Window struct {
	Range       string
	Granularity Granularity
	Start       time.Time
	End         time.Time
}

// Bucket is one x-axis point of a time series.
type Bucket struct {
	Start time.Time
	Label string
}

// SeriesPoint is one aggregated value inside a bucket.
type SeriesPoint struct {
	Bucket time.Time
	Value  int64
}

// HistoryEvent is one row of the consumption ledger.
type HistoryEvent struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"`
	KindLabel   string    `json:"kind_label"`
	Description string    `json:"description"`
	Amount      int64     `json:"amount"`
	Unit        string    `json:"unit"`
	CostIDR     int64     `json:"cost_idr"`
	CreatedAt   time.Time `json:"created_at"`
}

// SortField is one ordered column in a (possibly multi-column) sort.
type SortField struct {
	Column string
	Order  string // "asc" | "desc"
}

// HistoryFilter is the parsed, validated query for the history table.
type HistoryFilter struct {
	Kind   string
	Start  *time.Time
	End    *time.Time
	Search string
	// Sorts is the ordered list of sort columns. When empty the repository
	// falls back to created_at DESC.
	Sorts   []SortField
	Page    int
	PerPage int
}

// Repository reads a user's consumption. Every method is scoped to one user.
type Repository interface {
	// CountInBuckets returns, per kind, the aggregated value for each
	// granularity bucket in [start, end). Missing buckets are simply absent.
	CountInBuckets(ctx context.Context, userID uuid.UUID, kind string, g Granularity, start, end time.Time) ([]SeriesPoint, error)
	// History returns one page of consumption events plus the total row count.
	History(ctx context.Context, userID uuid.UUID, f HistoryFilter) ([]HistoryEvent, int, error)
}
