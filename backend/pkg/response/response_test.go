package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func performRequest(method, path string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	handler(c)
	return w
}

func TestJSON(t *testing.T) {
	w := performRequest("GET", "/test", func(c *gin.Context) {
		JSON(c, http.StatusOK, "success", map[string]string{"key": "value"})
	})

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if !resp.Success {
		t.Error("Expected Success = true")
	}
	if resp.Message != "success" {
		t.Errorf("Expected Message = 'success', got '%s'", resp.Message)
	}
	if resp.Data == nil {
		t.Error("Expected Data to not be nil")
	}
}

func TestError(t *testing.T) {
	w := performRequest("GET", "/error", func(c *gin.Context) {
		Error(c, http.StatusBadRequest, "bad request", "invalid_input")
	})

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Success {
		t.Error("Expected Success = false")
	}
	if resp.Message != "bad request" {
		t.Errorf("Expected Message = 'bad request', got '%s'", resp.Message)
	}
	if resp.Error != "invalid_input" {
		t.Errorf("Expected Error = 'invalid_input', got '%s'", resp.Error)
	}
}

func TestPaginated(t *testing.T) {
	w := performRequest("GET", "/paginated", func(c *gin.Context) {
		Paginated(c, "list", []string{"a", "b"}, 50, 1, 20)
	})

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if !resp.Success {
		t.Error("Expected Success = true")
	}
	if resp.Meta == nil {
		t.Fatal("Expected Meta to not be nil")
	}
	if resp.Meta.Total != 50 {
		t.Errorf("Expected Total = 50, got %d", resp.Meta.Total)
	}
	if resp.Meta.Page != 1 {
		t.Errorf("Expected Page = 1, got %d", resp.Meta.Page)
	}
	if resp.Meta.PerPage != 20 {
		t.Errorf("Expected PerPage = 20, got %d", resp.Meta.PerPage)
	}
	// total_pages must be derived so clients never compute NaN (50 / 20 -> 3).
	if resp.Meta.TotalPages != 3 {
		t.Errorf("Expected TotalPages = 3, got %d", resp.Meta.TotalPages)
	}
}

func TestPaginated_TotalPagesFloorsAtOne(t *testing.T) {
	// An empty result set must still report "page 1 of 1", never 0 or NaN.
	w := performRequest("GET", "/paginated", func(c *gin.Context) {
		Paginated(c, "list", []string{}, 0, 1, 20)
	})
	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Meta == nil || resp.Meta.TotalPages != 1 {
		t.Fatalf("expected TotalPages = 1 for an empty set, got %+v", resp.Meta)
	}
}

func TestStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		fn         func(c *gin.Context)
		wantStatus int
	}{
		{"OK", func(c *gin.Context) { OK(c, "ok", nil) }, http.StatusOK},
		{"Created", func(c *gin.Context) { Created(c, "created", nil) }, http.StatusCreated},
		{"BadRequest", func(c *gin.Context) { BadRequest(c, "bad", "err") }, http.StatusBadRequest},
		{"Unauthorized", func(c *gin.Context) { Unauthorized(c, "unauth") }, http.StatusUnauthorized},
		{"Forbidden", func(c *gin.Context) { Forbidden(c, "forbid") }, http.StatusForbidden},
		{"NotFound", func(c *gin.Context) { NotFound(c, "not found") }, http.StatusNotFound},
		{"InternalError", func(c *gin.Context) { InternalError(c, "internal") }, http.StatusInternalServerError},
		{"TooManyRequests", func(c *gin.Context) { TooManyRequests(c, "rate") }, http.StatusTooManyRequests},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := performRequest("GET", "/", tt.fn)
			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestSanitizeTechnicalDetails(t *testing.T) {
	w := performRequest("GET", "/500", func(c *gin.Context) {
		InternalError(c, "ERROR: column id of relation transactions does not exist (SQLSTATE 42703)")
	})

	var resp Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Message == "ERROR: column id of relation transactions does not exist (SQLSTATE 42703)" {
		t.Fatalf("technical SQL error leaked to client: %s", resp.Message)
	}
	if resp.Message != "Terjadi kesalahan internal pada server. Silakan coba beberapa saat lagi." {
		t.Errorf("expected sanitized message, got: %s", resp.Message)
	}
}

func TestSanitizeBadRequestTechnicalDetails(t *testing.T) {
	w := performRequest("POST", "/400", func(c *gin.Context) {
		BadRequest(c, "Invalid request body", "json: cannot unmarshal string into Go struct field Transaction.amount of type float64")
	})

	var resp Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if strings.Contains(resp.Error, "struct field") || strings.Contains(resp.Error, "cannot unmarshal") {
		t.Fatalf("technical unmarshal error leaked in error field: %s", resp.Error)
	}
	if resp.Error != "invalid_payload" {
		t.Errorf("expected error code invalid_payload, got: %s", resp.Error)
	}
	if resp.Message != "Format data yang dikirim tidak valid atau tidak lengkap." {
		t.Errorf("expected human friendly message, got: %s", resp.Message)
	}
}
