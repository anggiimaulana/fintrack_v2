package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

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
		Message: msg,
		Error:   err,
	})
}

func OK(c *gin.Context, msg string, data interface{}) {
	JSON(c, http.StatusOK, msg, data)
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
	Error(c, http.StatusBadRequest, msg, err)
}

func Unauthorized(c *gin.Context, msg string) {
	Error(c, http.StatusUnauthorized, msg, "unauthorized")
}

func Forbidden(c *gin.Context, msg string) {
	Error(c, http.StatusForbidden, msg, "forbidden")
}

func NotFound(c *gin.Context, msg string) {
	Error(c, http.StatusNotFound, msg, "not_found")
}

func InternalError(c *gin.Context, msg string) {
	Error(c, http.StatusInternalServerError, msg, "internal_error")
}

func Conflict(c *gin.Context, msg string) {
	Error(c, http.StatusConflict, msg, "conflict")
}

func TooManyRequests(c *gin.Context, msg string) {
	Error(c, http.StatusTooManyRequests, msg, "rate_limit_exceeded")
}
