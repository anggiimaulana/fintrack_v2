package validator

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/fintrack/backend/pkg/response"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func BindAndValidate(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return false
	}
	if err := validate.Struct(obj); err != nil {
		errs := make([]ValidationError, 0)
		for _, err := range err.(validator.ValidationErrors) {
			field := strings.ToLower(err.Field())
			message := getErrorMessage(err)
			errs = append(errs, ValidationError{Field: field, Message: message})
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Validation failed",
			"errors":  errs,
		})
		return false
	}
	return true
}

func BindQueryAndValidate(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindQuery(obj); err != nil {
		response.BadRequest(c, "Invalid query parameters", err.Error())
		return false
	}
	return true
}

func getErrorMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "This field is required"
	case "gt":
		return "Must be greater than " + err.Param()
	case "gte":
		return "Must be greater than or equal to " + err.Param()
	case "lt":
		return "Must be less than " + err.Param()
	case "lte":
		return "Must be less than or equal to " + err.Param()
	case "min":
		return "Minimum length is " + err.Param()
	case "max":
		return "Maximum length is " + err.Param()
	case "email":
		return "Invalid email format"
	case "oneof":
		return "Must be one of: " + err.Param()
	default:
		return "Invalid value"
	}
}

func SanitizeString(s string) string {
	var result strings.Builder
	for _, r := range s {
		if r == '<' || r == '>' || r == '"' || r == '\'' || r == '&' {
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

func SanitizeMap(data map[string]interface{}) {
	for k, v := range data {
		if str, ok := v.(string); ok {
			data[k] = SanitizeString(str)
		}
	}
}
