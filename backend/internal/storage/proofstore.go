package storage

import "context"

// ProofStore adapts the runtime-switchable Provider to the payment module's
// ProofStore contract (Save). It keeps the payment package free of any storage
// dependency while letting proof uploads follow the active driver.
type ProofStore struct {
	provider *Provider
}

// NewProofStore wraps a provider as a payment proof store.
func NewProofStore(p *Provider) *ProofStore { return &ProofStore{provider: p} }

// Save stores content under key and returns the same key. Callers persist the
// returned value as `payments.proof_url`; the API then serves it via
// GET /api/v1/payments/proofs/:id, which reads the active (or fallback) driver.
func (s *ProofStore) Save(ctx context.Context, key string, content []byte) (string, error) {
	store, err := s.provider.Active(ctx)
	if err != nil {
		return "", err
	}
	if err := store.Put(ctx, key, content, ContentTypeFor(key)); err != nil {
		return "", err
	}
	return key, nil
}

// Open reads a stored proof through the provider (with cross-driver fallback).
func (s *ProofStore) Open(ctx context.Context, key string) (*Object, error) {
	return s.provider.Open(ctx, key)
}

// Provider exposes the underlying provider (for status/handlers).
func (s *ProofStore) Provider() *Provider { return s.provider }
