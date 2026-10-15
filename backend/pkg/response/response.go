package response

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var technicalKeywords = []string{
	"sql:", "pq:", "pgx:", "select ", "update ", "insert ", "delete ", "from ", "where ",
	"violates", "constraint", "sqlstate", "dial tcp", "connect:", "connection refused",
	"syntax error", "column \"", "relation \"", "table \"", "panic", "goroutine", "runtime error",
	"unmarshal", "cannot unmarshal", "invalid character", "parsing time", "struct field",
	"failed on the '", "strconv.", "invalid uuid", "json:", "unexpected eof",
}

func isTechnical(s string) bool {
	if s == "" {
		return false
	}
	lower := strings.ToLower(s)
	for _, kw := range technicalKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return strings.Contains(s, "\n") || strings.HasPrefix(lower, "error:")
}

func sanitizeMessage(status int, msg string) string {
	if isTechnical(msg) {
		if status >= 500 {
			return "Terjadi kesalahan internal pada server. Silakan coba beberapa saat lagi."
		}
		return "Format data yang dikirim tidak valid atau tidak sesuai."
	}
	if msg == "" && status >= 500 {
		return "Terjadi kesalahan internal pada server. Silakan coba beberapa saat lagi."
	}
	return msg
}

func sanitizeErrorCode(status int, errCode string) string {
	if status >= 500 {
		return "internal_error"
	}
	if isTechnical(errCode) {
		return "invalid_payload"
	}
	return errCode
}

type Meta struct {
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

func JSON(c *gin.Context, status int, msg string, data interface{}) {
	c.JSON(status, Response{
		Success: status >= 200 && status < 300,
		Message: msg,
		Data:    data,
	})
}

func Error(c *gin.Context, status int, msg string, err string) {
	c.JSON(status, Response{
		Success: false,
		Message: sanitizeMessage(status, msg),
		Error:   sanitizeErrorCode(status, err),
	})
}

func OK(c *gin.Context, msg string, data interface{}) {
	JSON(c, http.StatusOK, msg, data)
}

// Envelope builds the standard success body without writing it. Useful for
// caching the serialised payload keyed on the response shape.
func Envelope(msg string, data interface{}) Response {
	return Response{Success: true, Message: msg, Data: data}
}

func Created(c *gin.Context, msg string, data interface{}) {
	JSON(c, http.StatusCreated, msg, data)
}

func Paginated(c *gin.Context, msg string, data interface{}, total, page, perPage int) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: msg,
		Data:    data,
		Meta: &Meta{
			Total:   total,
			Page:    page,
			PerPage: perPage,
		},
	})
}

func BadRequest(c *gin.Context, msg string, err string) {
	if msg == "Invalid request body" || msg == "Invalid request" || msg == "" {
		msg = "Format data yang dikirim tidak valid atau tidak lengkap."
	}
	Error(c, http.StatusBadRequest, msg, err)
}

func Unauthorized(c *gin.Context, msg string) {
	if msg == "" {
		msg = "Autentikasi diperlukan untuk mengakses layanan ini."
	}
	Error(c, http.StatusUnauthorized, msg, "unauthorized")
}

func Forbidden(c *gin.Context, msg string) {
	if msg == "" {
		msg = "Anda tidak memiliki izin untuk melakukan tindakan ini."
	}
	Error(c, http.StatusForbidden, msg, "forbidden")
}

func NotFound(c *gin.Context, msg string) {
	if msg == "" {
		msg = "Data atau sumber daya yang diminta tidak ditemukan."
	}
	Error(c, http.StatusNotFound, msg, "not_found")
}

func InternalError(c *gin.Context, msg string) {
	if msg == "" {
		msg = "Terjadi kesalahan internal pada server. Silakan coba beberapa saat lagi."
	}
	Error(c, http.StatusInternalServerError, msg, "internal_error")
}

func Conflict(c *gin.Context, msg string) {
	if msg == "" {
		msg = "Data yang dikirim bertentangan dengan data yang sudah ada."
	}
	Error(c, http.StatusConflict, msg, "conflict")
}

func TooManyRequests(c *gin.Context, msg string) {
	if msg == "" || msg == "Too many requests" {
		msg = "Terlalu banyak permintaan. Silakan tunggu beberapa saat."
	}
	Error(c, http.StatusTooManyRequests, msg, "rate_limit_exceeded")
}

// QuotaExceeded answers 429 when a user's token quota cannot cover a request.
// It uses the same envelope shape as the plan-gate middleware (feature/limit
// gates) so the frontend has one code path for "upgrade to continue": a
// machine-readable error code plus the upgrade hint and URL.
func QuotaExceeded(c *gin.Context, msg string) {
	if msg == "" {
		msg = "Sisa kuota Anda tidak mencukupi untuk permintaan ini."
	}
	c.JSON(http.StatusTooManyRequests, gin.H{
		"success":      false,
		"error":        "insufficient_quota",
		"message":      msg,
		"upgrade_hint": true,
		"upgrade_url":  "/user/subscription",
	})
}
