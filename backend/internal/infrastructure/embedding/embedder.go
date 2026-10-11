package embedding

import (
	"context"
	"errors"
)

// ErrInvalidDimension is returned when a produced vector is not 768-dimensional.
// The pgvector column is fixed at vector(768); a wrong dimension would corrupt
// the index, so the check is enforced at runtime.
var ErrInvalidDimension = errors.New("invalid embedding dimension: expected 768")

// ExpectedDimensions is the locked embedding size for FinTrack (text-embedding-004).
const ExpectedDimensions = 768

// Embedder is the minimal surface used by business code for RAG. Both the
// legacy direct Gemini embedder and the gateway-backed embedder satisfy it.
type Embedder interface {
	// Embed returns the vector for text.
	Embed(ctx context.Context, text string) ([]float32, error)
	// Enabled reports whether embedding is available.
	Enabled() bool
}

// validateDimensions enforces the 768-dimension lock.
func validateDimensions(vec []float32) error {
	if len(vec) != ExpectedDimensions {
		return ErrInvalidDimension
	}
	return nil
}

// compile-time assertion for the legacy embedder.
var _ Embedder = (*GeminiEmbedder)(nil)
