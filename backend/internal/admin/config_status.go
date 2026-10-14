package admin

import (
	"github.com/gin-gonic/gin"

	"github.com/fintrack/backend/internal/runtimeconfig"
	"github.com/fintrack/backend/pkg/response"
)

// ConfigStatus handles GET /admin/settings/config. It reports the *effective*
// value of runtime-resolved configuration and, for secrets, whether a value is
// set (never the value itself). The UI uses this to show which credential is
// active and where it comes from (DB override vs env default).
func (h *Handler) ConfigStatus(c *gin.Context) {
	ctx := c.Request.Context()
	res := h.svc.runtimeResolver()
	// Effective (non-secret) values.
	effective := gin.H{
		"midtrans_enabled":       res.Bool(ctx, "midtrans_enabled", "MIDTRANS_ENABLED", false),
		"midtrans_is_production": res.Bool(ctx, "midtrans_is_production", "MIDTRANS_IS_PRODUCTION", false),
		"midtrans_client_key":    res.String(ctx, "midtrans_client_key", "MIDTRANS_CLIENT_KEY", ""),
		"manual_bank_name":       res.String(ctx, "manual_bank_name", "MANUAL_BANK_NAME", ""),
		"manual_account_number":  res.String(ctx, "manual_account_number", "MANUAL_ACCOUNT_NUMBER", ""),
		"manual_account_holder":  res.String(ctx, "manual_account_holder", "MANUAL_ACCOUNT_HOLDER", ""),
		"r2_account_id":          res.String(ctx, "r2_account_id", "R2_ACCOUNT_ID", ""),
		"r2_bucket":              res.String(ctx, "r2_bucket", "R2_BUCKET", ""),
		"r2_region":              res.String(ctx, "r2_region", "R2_REGION", "auto"),
		"r2_endpoint":            res.String(ctx, "r2_endpoint", "R2_ENDPOINT", ""),
		"r2_prefix":              res.String(ctx, "r2_prefix", "R2_PREFIX", ""),
		"r2_public_base_url":     res.String(ctx, "r2_public_base_url", "R2_PUBLIC_BASE_URL", ""),
		"ai_reasoning_effort":    res.String(ctx, "ai_reasoning_effort", "AI_REASONING_EFFORT", ""),
		"mail_enabled":           res.Bool(ctx, "mail_enabled", "MAIL_ENABLED", false),
		"mail_host":              res.String(ctx, "mail_host", "SMTP_HOST", ""),
	}

	// Secret presence + a masked hint of the active value.
	secrets := gin.H{
		"midtrans_server_key":      res.MaskSecret(ctx, "midtrans_server_key", "MIDTRANS_SERVER_KEY"),
		"r2_secret_access_key":     res.MaskSecret(ctx, "r2_secret_access_key", "R2_SECRET_ACCESS_KEY"),
		"mail_password":            res.MaskSecret(ctx, "mail_password", "SMTP_PASSWORD"),
		"gemini_embedding_api_key": res.MaskSecret(ctx, "gemini_embedding_api_key", "GEMINI_EMBEDDING_API_KEY"),
	}
	hasSecret := gin.H{
		"midtrans_server_key":      res.HasSecret(ctx, "midtrans_server_key", "MIDTRANS_SERVER_KEY"),
		"r2_secret_access_key":     res.HasSecret(ctx, "r2_secret_access_key", "R2_SECRET_ACCESS_KEY"),
		"mail_password":            res.HasSecret(ctx, "mail_password", "SMTP_PASSWORD"),
		"gemini_embedding_api_key": res.HasSecret(ctx, "gemini_embedding_api_key", "GEMINI_EMBEDDING_API_KEY"),
	}

	// Which source supplied each secret (DB override vs environment).
	source := gin.H{
		"midtrans_server_key":      res.SourceOf(ctx, "midtrans_server_key", "MIDTRANS_SERVER_KEY"),
		"r2_secret_access_key":     res.SourceOf(ctx, "r2_secret_access_key", "R2_SECRET_ACCESS_KEY"),
		"mail_password":            res.SourceOf(ctx, "mail_password", "SMTP_PASSWORD"),
		"gemini_embedding_api_key": res.SourceOf(ctx, "gemini_embedding_api_key", "GEMINI_EMBEDDING_API_KEY"),
	}

	response.OK(c, "Status konfigurasi berhasil dimuat", gin.H{
		"effective":  effective,
		"secrets":    secrets,
		"has_secret": hasSecret,
		"source":     source,
	})
}

// runtimeResolver exposes the resolver, constructing an env-only one if the
// service has no settings repo (defensive; not expected in production).
func (s *Service) runtimeResolver() *runtimeconfig.Resolver {
	if s.runtime != nil {
		return s.runtime
	}
	return runtimeconfig.New(nil, s.encKey)
}

// WithRuntimeResolver installs the shared runtime resolver.
func WithRuntimeResolver(r *runtimeconfig.Resolver) Option {
	return func(s *Service) { s.runtime = r }
}
