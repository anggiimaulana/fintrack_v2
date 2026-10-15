package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/fintrack/backend/internal/timeouts"
	"github.com/fintrack/backend/pkg/response"
)

// AIConfigHandler exposes the effective AI timeout budget to the browser.
//
// The client needs to know how long it may wait for an AI request before it
// aborts; if that budget were hardcoded, a Super Admin widening the server-side
// timeout would still see the browser give up first. Reading it from here keeps
// the two ends consistent, and because the underlying settings are dynamic the
// value follows a Super Admin change without a redeploy.
type AIConfigHandler struct {
	reader timeouts.ConfigReader
}

// NewAIConfigHandler builds the handler. A nil reader yields the defaults.
func NewAIConfigHandler(reader timeouts.ConfigReader) *AIConfigHandler {
	return &AIConfigHandler{reader: reader}
}

// Get handles GET /me/ai-config.
func (h *AIConfigHandler) Get(c *gin.Context) {
	cfg := timeouts.Load(c.Request.Context(), h.reader)
	response.OK(c, "Konfigurasi AI berhasil dimuat", gin.H{
		// Milliseconds the client should allow for an AI request. 0 means "no
		// explicit budget" — the client falls back to its built-in default.
		"request_timeout_ms": cfg.Client.Milliseconds(),
		// The outer ceiling imposed by the Next.js rewrite proxy. Every budget is
		// clamped to it, so the UI can explain why a larger value is ineffective
		// instead of silently ignoring it.
		"proxy_ceiling_ms": timeouts.ProxyCeilingMS(),
		// Per-use-case budgets, for diagnostics and any client that wants to
		// show or reason about them.
		"timeouts_ms": gin.H{
			"chat":      cfg.Chat.Milliseconds(),
			"vision":    cfg.Vision.Milliseconds(),
			"voice":     cfg.Voice.Milliseconds(),
			"embedding": cfg.Embedding.Milliseconds(),
			"reasoning": cfg.Reasoning.Milliseconds(),
		},
	})
}
