package admin

import (
	"github.com/gin-gonic/gin"

	"github.com/fintrack/backend/internal/entitlement"
	"github.com/fintrack/backend/pkg/response"
)

// entitlementCatalogResponse is the payload for GET /admin/entitlement-catalog.
type entitlementCatalogResponse struct {
	Features []entitlement.FeatureKey `json:"features"`
	Limits   []entitlement.LimitKey   `json:"limits"`
}

// EntitlementCatalog handles GET /admin/entitlement-catalog. It exposes the
// authoritative feature/limit key registry so the plan editor can offer a
// dropdown instead of free-text input — eliminating typo'd keys (F5).
func (h *Handler) EntitlementCatalog(c *gin.Context) {
	response.OK(c, "Katalog entitlement", entitlementCatalogResponse{
		Features: entitlement.FeatureCatalog,
		Limits:   entitlement.LimitCatalog,
	})
}
