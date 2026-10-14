package embedding

import (
	"context"

	"github.com/fintrack/backend/internal/domain"
)

// GatewayEmbedder produces embeddings through the centralized AI gateway
// (P03), keeping all AI traffic on the pooled endpoints. It enforces the
// 768-dimension lock on the returned vector.
type GatewayEmbedder struct {
	gateway domain.AIGatewayService
	// model, when non-empty, overrides the model configured on the embedding
	// route. It is intentionally left empty by default so the Super Admin owns
	// the model choice through the `ai_routes` table (no hardcoding).
	model string
}

// NewGatewayEmbedder builds a gateway-backed embedder. The upstream model is
// resolved from the active `embedding` route, so it stays configurable from the
// Super Admin UI.
func NewGatewayEmbedder(gateway domain.AIGatewayService) *GatewayEmbedder {
	return &GatewayEmbedder{gateway: gateway}
}

// Enabled reports whether the gateway is configured.
func (e *GatewayEmbedder) Enabled() bool { return e != nil && e.gateway != nil }

// Embed returns the 768-dim vector for text via the gateway.
func (e *GatewayEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if !e.Enabled() {
		return nil, ErrDisabled
	}
	vec, _, err := e.gateway.CreateEmbedding(ctx, &domain.AIEmbeddingRequest{
		Model: e.model,
		Text:  text,
	})
	if err != nil {
		return nil, err
	}
	if err := validateDimensions(vec); err != nil {
		return nil, err
	}
	return vec, nil
}

// compile-time assertion.
var _ Embedder = (*GatewayEmbedder)(nil)
