// Package pagination parses table query parameters (page, per_page, sort, order,
// search, filters) into a safe, validated form and builds the response
// envelope used by every list endpoint. Sorting columns are validated against a
// per-endpoint whitelist to prevent SQL injection via ORDER BY.
package pagination

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Limits for per_page.
const (
	DefaultPerPage = 20
	MinPerPage     = 10
	MaxPerPage     = 100
	DefaultPage    = 1
)

// Options customises parsing per endpoint.
type Options struct {
	// AllowedSort is the whitelist of sortable column names. When empty, sort
	// is ignored entirely.
	AllowedSort []string
	// DefaultSort is applied when no (valid) sort is provided.
	DefaultSort string
	// DefaultOrder is "asc" or "desc" (default "desc").
	DefaultOrder string
	// MaxPerPage overrides the global cap when > 0.
	MaxPerPage int
}

// SortSpec is one column and its direction within a (possibly multi-column)
// sort. `Query.Sort`/`Query.Order` mirror the first entry for callers that only
// understand a single column.
type SortSpec struct {
	Column string
	Order  string // "asc" | "desc"
}

// Query is the parsed, validated table query.
type Query struct {
	Page    int
	PerPage int
	Sort    string
	Order   string // "asc" | "desc"
	Search  string
	Filters map[string]string
	// Sorts is the full ordered sort list, parsed from
	// `sort=a,b&order=desc,asc`. Empty when sorting is disabled or no column is
	// valid. Callers that support multi-column sort should prefer
	// OrderByClauses over OrderByClause.
	Sorts []SortSpec
}

// Offset returns the SQL OFFSET for the current page.
func (q Query) Offset() int { return (q.Page - 1) * q.PerPage }

// Parse reads the table query parameters from the request:
//
//	?page=2&per_page=20&sort=created_at&order=desc&q=text&filter_status=active
//
// Invalid values fall back to safe defaults. Sort columns not present in
// opts.AllowedSort are ignored.
func Parse(c *gin.Context, opts Options) Query {
	q := Query{
		Page:    DefaultPage,
		PerPage: DefaultPerPage,
		Order:   normalizeOrder(opts.DefaultOrder),
		Filters: map[string]string{},
	}

	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			q.Page = n
		}
	}

	maxPer := MaxPerPage
	if opts.MaxPerPage > 0 {
		maxPer = opts.MaxPerPage
	}
	if v := c.Query("per_page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.PerPage = clamp(n, MinPerPage, maxPer)
		}
	}

	// Sort must be whitelisted.
	if sort := c.Query("sort"); sort != "" && allowed(opts.AllowedSort, sort) {
		q.Sort = sort
	} else if opts.DefaultSort != "" && allowed(opts.AllowedSort, opts.DefaultSort) {
		q.Sort = opts.DefaultSort
	}

	if order := c.Query("order"); order != "" {
		q.Order = normalizeOrder(order)
	}

	// Multi-column sort (additive to the single-column fields above so existing
	// callers keep working unchanged).
	q.Sorts = parseSorts(c, opts)

	q.Search = strings.TrimSpace(c.Query("q"))

	// Filter parameters: filter_<key>=value
	for k, v := range c.Request.URL.Query() {
		if strings.HasPrefix(k, "filter_") && len(v) > 0 && v[0] != "" {
			key := strings.TrimPrefix(k, "filter_")
			q.Filters[key] = v[0]
		}
	}

	return q
}

// OrderByClause returns a safe "col ORDER" fragment (empty when sort invalid).
// The column is guaranteed to be in the whitelist; the direction is asc/desc.
func OrderByClause(q Query, whitelist []string) string {
	if q.Sort == "" || !allowed(whitelist, q.Sort) {
		return ""
	}
	return q.Sort + " " + strings.ToUpper(q.Order)
}

// parseSorts reads a comma-separated sort list (`sort=a,b&order=desc,asc`),
// keeping only whitelisted columns and normalising every direction. When no
// valid column is supplied it falls back to the endpoint's DefaultSort. A
// single `sort=a` yields exactly one entry, so this is a superset of the
// single-column behaviour.
func parseSorts(c *gin.Context, opts Options) []SortSpec {
	raw := strings.TrimSpace(c.Query("sort"))
	def := normalizeOrder(opts.DefaultOrder)
	if raw == "" {
		if opts.DefaultSort != "" && allowed(opts.AllowedSort, opts.DefaultSort) {
			return []SortSpec{{Column: opts.DefaultSort, Order: def}}
		}
		return nil
	}

	cols := strings.Split(raw, ",")
	ords := strings.Split(strings.TrimSpace(c.Query("order")), ",")
	out := make([]SortSpec, 0, len(cols))
	for i, col := range cols {
		col = strings.TrimSpace(col)
		if col == "" || !allowed(opts.AllowedSort, col) {
			continue
		}
		order := def
		if i < len(ords) {
			if o := strings.TrimSpace(ords[i]); o != "" {
				order = normalizeOrder(o)
			}
		} else if n := len(ords); n > 0 {
			// Fewer directions than columns: reuse the last explicit one.
			if o := strings.TrimSpace(ords[n-1]); o != "" {
				order = normalizeOrder(o)
			}
		}
		out = append(out, SortSpec{Column: col, Order: order})
	}

	if len(out) == 0 && opts.DefaultSort != "" && allowed(opts.AllowedSort, opts.DefaultSort) {
		out = append(out, SortSpec{Column: opts.DefaultSort, Order: def})
	}
	return out
}

// OrderByClauses returns a safe, comma-joined "col ORDER, col ORDER" fragment
// built from Query.Sorts (each column validated against the whitelist). It
// degrades to OrderByClause when no multi-sort was parsed, so it is a drop-in
// replacement.
func OrderByClauses(q Query, whitelist []string) string {
	if len(q.Sorts) == 0 {
		return OrderByClause(q, whitelist)
	}
	parts := make([]string, 0, len(q.Sorts))
	for _, s := range q.Sorts {
		if !allowed(whitelist, s.Column) {
			continue
		}
		parts = append(parts, s.Column+" "+strings.ToUpper(normalizeOrder(s.Order)))
	}
	return strings.Join(parts, ", ")
}

func normalizeOrder(order string) string {
	if strings.EqualFold(order, "asc") {
		return "asc"
	}
	return "desc"
}

func allowed(list []string, col string) bool {
	for _, c := range list {
		if c == col {
			return true
		}
	}
	return false
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
