package embedding

import (
	"context"
	"errors"
)

// ErrKillSwitch is returned when an operator stopped the embedding modality.
var ErrKillSwitch = errors.New("embeddings disabled by kill switch")

// KillSwitchEmbedder decorates any Embedder with the admin emergency stop, so
// the "embedding" switch also covers transports that do not consult it
// themselves (the direct Gemini embedder). Chat already fails closed earlier,
// this keeps the async indexing worker from spending tokens while stopped.
type KillSwitchEmbedder struct {
	inner   Embedder
	stopped func() bool
}

// NewKillSwitchEmbedder wraps inner; stopped reports the live switch state.
func NewKillSwitchEmbedder(inner Embedder, stopped func() bool) *KillSwitchEmbedder {
	return &KillSwitchEmbedder{inner: inner, stopped: stopped}
}

// Embed refuses while the switch is on, otherwise delegates.
func (k *KillSwitchEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if k.stopped != nil && k.stopped() {
		return nil, ErrKillSwitch
	}
	return k.inner.Embed(ctx, text)
}

// Enabled reports availability, false while stopped.
func (k *KillSwitchEmbedder) Enabled() bool {
	return k.inner.Enabled() && (k.stopped == nil || !k.stopped())
}
