package admin

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/mail"
	"github.com/fintrack/backend/pkg/response"
)

// SendTestEmail handles POST /admin/settings/mail/test. It sends a test message
// to the given address using the *effective* configuration (DB setting over env
// default), so an operator can verify SMTP before relying on it. The address
// defaults to the configured From when omitted.
func (h *Handler) SendTestEmail(c *gin.Context) {
	var req struct {
		To string `json:"to"`
	}
	_ = c.ShouldBindJSON(&req)

	provider := h.svc.mailProvider()
	if provider == nil {
		response.InternalError(c, "Konfigurasi email tidak tersedia")
		return
	}

	ctx := c.Request.Context()
	cfg := provider.Config(ctx)
	if !cfg.Enabled || cfg.Host == "" || cfg.From == "" {
		response.Error(c, 409, "Email belum diaktifkan atau belum lengkap. Isi host, port, dan pengirim terlebih dahulu.", "mail_disabled")
		return
	}

	to := strings.TrimSpace(req.To)
	if to == "" {
		to = cfg.From
	}

	sender := provider.Sender(ctx)
	if err := sender.Send(ctx, to, "FinTrack — Tes Konfigurasi Email",
		"Ini adalah email uji dari FinTrack. Jika Anda menerimanya, konfigurasi SMTP sudah benar."); err != nil {
		// 422 (not 5xx) keeps the message intact: sanitizeMessage() replaces any
		// 5xx body with a generic string, which would hide the SMTP failure the
		// admin needs to act on. The detail is a mail error, not an internal fault.
		h.audit(c, "test", "settings", "mail", nil, gin.H{"to": to, "ok": false})
		response.Error(c, 422, "Gagal mengirim email uji: "+err.Error(), "mail_failed")
		return
	}

	h.audit(c, "test", "settings", "mail", nil, gin.H{"to": to, "ok": true})
	response.OK(c, "Email uji berhasil dikirim ke "+to, gin.H{"to": to})
}

// MailStatus handles GET /admin/settings/mail — a read-only view of the
// effective mail configuration with the password redacted, so the UI can show
// whether email is active and which source (DB/env) is in effect.
func (h *Handler) MailStatus(c *gin.Context) {
	provider := h.svc.mailProvider()
	if provider == nil {
		response.OK(c, "Konfigurasi email belum tersedia", gin.H{"enabled": false, "configured": false})
		return
	}
	cfg := provider.Config(c.Request.Context())
	response.OK(c, "Status email berhasil dimuat", gin.H{
		"enabled":      cfg.Enabled,
		"configured":   provider.Sender(c.Request.Context()).Configured(),
		"host":         cfg.Host,
		"port":         cfg.Port,
		"username":     cfg.Username,
		"from":         cfg.From,
		"from_name":    cfg.FromName,
		"use_tls":      cfg.UseTLS,
		"has_password": cfg.Password != "",
	})
}

// mailProvider builds a mail.Provider from the service's settings pool + key.
// Kept on Service so handlers do not need their own dependency.
func (s *Service) mailProvider() *mail.Provider {
	if s.pool == nil {
		return nil
	}
	return mail.NewProvider(&settingsReader{pool: s.pool}, s.encKey)
}

// settingsReader adapts the pool to mail.SettingReader without importing the
// postgres repository (avoids an import cycle and keeps the adapter tiny).
type settingsReader struct{ pool *pgxpool.Pool }

func (r *settingsReader) GetString(ctx context.Context, key string) (string, error) {
	var raw []byte
	err := r.pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = $1`, key).Scan(&raw)
	if err != nil {
		return "", nil // absent -> empty, not an error
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return string(raw), nil
	}
	return s, nil
}
