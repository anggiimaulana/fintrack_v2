package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/pkg/response"
)

// GeminiKeyHandler is DEPRECATED as of P03. FinTrack now routes all AI traffic
// through the centralized AI gateway, so per-user BYOK key management is no
// longer available.
//
// Behaviour (per TASK-P03-011):
//   - Mutations (POST/PUT/PATCH/DELETE): HTTP 410 Gone with a machine-readable
//     error code so legacy clients fail clearly.
//   - List (GET): HTTP 200 with an empty array, so old clients that still poll
//     the endpoint do not crash.
type GeminiKeyHandler struct {
	usecase *usecase.GeminiKeyUsecase
}

func NewGeminiKeyHandler(usecase *usecase.GeminiKeyUsecase) *GeminiKeyHandler {
	return &GeminiKeyHandler{usecase: usecase}
}

// byokGone writes the standard deprecation payload.
func byokGone(c *gin.Context) {
	c.JSON(410, gin.H{
		"success": false,
		"status":  410,
		"error":   "BYOK_DEPRECATED",
		"message": "Fitur API Key mandiri telah dihentikan. FinTrack v2 kini menggunakan AI Gateway terpusat.",
	})
}

// List returns an empty array to keep legacy clients stable.
func (h *GeminiKeyHandler) List(c *gin.Context) {
	response.OK(c, "API keys retrieved successfully", []any{})
}

// Create is deprecated.
func (h *GeminiKeyHandler) Create(c *gin.Context) { byokGone(c) }

// Update is deprecated.
func (h *GeminiKeyHandler) Update(c *gin.Context) { byokGone(c) }

// Delete is deprecated.
func (h *GeminiKeyHandler) Delete(c *gin.Context) { byokGone(c) }

// Test is deprecated.
func (h *GeminiKeyHandler) Test(c *gin.Context) { byokGone(c) }
