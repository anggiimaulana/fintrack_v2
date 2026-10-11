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

// Query is the parsed, validated table query.
type Query struct {
	Page    int
	PerPage int
	Sort    string
	Order   string // "asc" | "desc"
	Search  string
	Filters map[string]string
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
