package pagination

import "math"

// Meta is the pagination envelope returned alongside list data.
type Meta struct {
	Total      int `json:"total"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}

// NewMeta builds a Meta from the query and total row count.
func NewMeta(q Query, total int) Meta {
	totalPages := 0
	if q.PerPage > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(q.PerPage)))
	}
	return Meta{
		Total:      total,
		Page:       q.Page,
		PerPage:    q.PerPage,
		TotalPages: totalPages,
	}
}
