package pagination

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
)

func newCtx(rawQuery string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/x?"+rawQuery, nil)
	return c
}

var sortWhitelist = []string{"created_at", "amount", "name"}

// TestParse_Defaults asserts safe defaults when no params are provided.
func TestParse_Defaults(t *testing.T) {
	q := Parse(newCtx(""), Options{AllowedSort: sortWhitelist, DefaultSort: "created_at"})
	if q.Page != 1 || q.PerPage != 20 {
		t.Fatalf("expected page=1 per_page=20, got %d/%d", q.Page, q.PerPage)
	}
	if q.Sort != "created_at" || q.Order != "desc" {
		t.Fatalf("expected default sort created_at desc, got %s %s", q.Sort, q.Order)
	}
	if q.Offset() != 0 {
		t.Fatalf("expected offset 0, got %d", q.Offset())
	}
}

// TestParse_ClampsPerPage asserts per_page is clamped to [10,100].
func TestParse_ClampsPerPage(t *testing.T) {
	if q := Parse(newCtx("per_page=5"), Options{}); q.PerPage != 10 {
		t.Fatalf("per_page=5 should clamp to 10, got %d", q.PerPage)
	}
	if q := Parse(newCtx("per_page=500"), Options{}); q.PerPage != 100 {
		t.Fatalf("per_page=500 should clamp to 100, got %d", q.PerPage)
	}
	if q := Parse(newCtx("per_page=50"), Options{}); q.PerPage != 50 {
		t.Fatalf("per_page=50 should stay 50, got %d", q.PerPage)
	}
}

// TestParse_RejectsInvalidPage asserts non-positive/garbage page falls back to 1.
func TestParse_RejectsInvalidPage(t *testing.T) {
	for _, v := range []string{"0", "-3", "abc"} {
		if q := Parse(newCtx("page="+v), Options{}); q.Page != 1 {
			t.Fatalf("page=%s should fall back to 1, got %d", v, q.Page)
		}
	}
}

// TestParse_SortWhitelist asserts only whitelisted columns are accepted.
func TestParse_SortWhitelist(t *testing.T) {
	q := Parse(newCtx("sort=amount&order=asc"), Options{AllowedSort: sortWhitelist, DefaultSort: "created_at"})
	if q.Sort != "amount" || q.Order != "asc" {
		t.Fatalf("expected amount asc, got %s %s", q.Sort, q.Order)
	}

	// Injection attempt / unknown column is ignored -> falls back to default.
	q = Parse(newCtx("sort="+url.QueryEscape("id;DROP TABLE users--")), Options{AllowedSort: sortWhitelist, DefaultSort: "created_at"})
	if q.Sort != "created_at" {
		t.Fatalf("unsafe sort must be ignored, got %q", q.Sort)
	}
}

// TestParse_OrderNormalization asserts order only ever becomes asc/desc.
func TestParse_OrderNormalization(t *testing.T) {
	if q := Parse(newCtx("order=ASC"), Options{}); q.Order != "asc" {
		t.Fatalf("expected asc, got %s", q.Order)
	}
	if q := Parse(newCtx("order=weird"), Options{}); q.Order != "desc" {
		t.Fatalf("expected desc fallback, got %s", q.Order)
	}
}

// TestParse_SearchAndFilters asserts q and filter_* parsing.
func TestParse_SearchAndFilters(t *testing.T) {
	q := Parse(newCtx("q=%20makan%20&filter_status=active&filter_type=expense"), Options{})
	if q.Search != "makan" {
		t.Fatalf("expected trimmed search 'makan', got %q", q.Search)
	}
	if q.Filters["status"] != "active" || q.Filters["type"] != "expense" {
		t.Fatalf("expected filters parsed, got %v", q.Filters)
	}
}

// TestOffset asserts offset math.
func TestOffset(t *testing.T) {
	q := Parse(newCtx("page=3&per_page=20"), Options{})
	if q.Offset() != 40 {
		t.Fatalf("expected offset 40, got %d", q.Offset())
	}
}

// TestOrderByClause asserts a safe ORDER BY fragment.
func TestOrderByClause(t *testing.T) {
	q := Parse(newCtx("sort=amount&order=asc"), Options{AllowedSort: sortWhitelist})
	if got := OrderByClause(q, sortWhitelist); got != "amount ASC" {
		t.Fatalf("expected 'amount ASC', got %q", got)
	}
	// Unknown column yields empty clause.
	bad := Query{Sort: "evil", Order: "asc"}
	if got := OrderByClause(bad, sortWhitelist); got != "" {
		t.Fatalf("expected empty clause for unsafe sort, got %q", got)
	}
}

// TestParse_MultiSort asserts comma-separated sort/order parse into Sorts and
// that unknown columns are dropped.
func TestParse_MultiSort(t *testing.T) {
	q := Parse(newCtx("sort=amount,name&order=asc,desc"), Options{AllowedSort: sortWhitelist, DefaultSort: "created_at"})
	if len(q.Sorts) != 2 {
		t.Fatalf("expected 2 sorts, got %d (%+v)", len(q.Sorts), q.Sorts)
	}
	if q.Sorts[0].Column != "amount" || q.Sorts[0].Order != "asc" {
		t.Fatalf("unexpected first sort: %+v", q.Sorts[0])
	}
	if q.Sorts[1].Column != "name" || q.Sorts[1].Order != "desc" {
		t.Fatalf("unexpected second sort: %+v", q.Sorts[1])
	}

	// A non-whitelisted column is silently dropped, the valid one survives.
	q = Parse(newCtx("sort=amount,evil&order=asc,desc"), Options{AllowedSort: sortWhitelist})
	if len(q.Sorts) != 1 || q.Sorts[0].Column != "amount" {
		t.Fatalf("expected only amount to survive, got %+v", q.Sorts)
	}

	// Single column still yields exactly one entry (superset of old behaviour).
	q = Parse(newCtx("sort=amount&order=asc"), Options{AllowedSort: sortWhitelist})
	if len(q.Sorts) != 1 || q.Sorts[0].Column != "amount" || q.Sorts[0].Order != "asc" {
		t.Fatalf("unexpected single sort: %+v", q.Sorts)
	}
}

// TestOrderByClauses asserts the multi-column fragment and its fallback.
func TestOrderByClauses(t *testing.T) {
	q := Parse(newCtx("sort=amount,name&order=asc,desc"), Options{AllowedSort: sortWhitelist})
	if got := OrderByClauses(q, sortWhitelist); got != "amount ASC, name DESC" {
		t.Fatalf("expected 'amount ASC, name DESC', got %q", got)
	}

	// No Sorts falls back to the single-column clause.
	fallback := Query{Sort: "amount", Order: "desc"}
	if got := OrderByClauses(fallback, sortWhitelist); got != "amount DESC" {
		t.Fatalf("expected fallback 'amount DESC', got %q", got)
	}
}

// TestNewMeta asserts total-page math.
func TestNewMeta(t *testing.T) {
	q := Query{Page: 2, PerPage: 20}
	m := NewMeta(q, 45)
	if m.Total != 45 || m.Page != 2 || m.PerPage != 20 || m.TotalPages != 3 {
		t.Fatalf("unexpected meta: %+v", m)
	}
	if NewMeta(Query{PerPage: 20}, 0).TotalPages != 0 {
		t.Fatal("zero rows must yield zero pages")
	}
}
