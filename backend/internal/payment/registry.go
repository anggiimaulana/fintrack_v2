package payment

import (
	"context"
	"fmt"
	"sync"
)

// Registry holds the configured gateways and resolves which one to use for a
// checkout (global default, per-plan override, then emergency fallback).
type Registry struct {
	mu       sync.RWMutex
	gateways map[string]Gateway
}

// NewRegistry builds an empty registry.
func NewRegistry() *Registry {
	return &Registry{gateways: map[string]Gateway{}}
}

// Register adds/replaces a gateway implementation.
func (r *Registry) Register(g Gateway) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gateways[g.Name()] = g
}

// Get returns a gateway by name.
func (r *Registry) Get(name string) (Gateway, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	g, ok := r.gateways[name]
	return g, ok
}

// Names lists registered gateway names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.gateways))
	for k := range r.gateways {
		out = append(out, k)
	}
	return out
}

// PlanGatewayResolver returns the per-plan gateway override ("" = none).
type PlanGatewayResolver func(ctx context.Context, planCode string) (string, error)

// Availability lets a gateway report whether it is usable right now (e.g.
// Midtrans with no server key). Checked before a gateway is selected so a
// disabled provider never breaks checkout.
type Availability interface {
	Enabled(ctx context.Context) bool
}

// available reports whether a gateway is usable. Gateways without an
// Availability implementation are always considered available.
func available(ctx context.Context, g Gateway) bool {
	if a, ok := g.(Availability); ok {
		return a.Enabled(ctx)
	}
	return true
}

// ResolveGateway selects a gateway: per-plan override > global default setting >
// emergency fallback ("manual").
//
// The chosen gateway name is locked onto the payment row at checkout, so
// in-flight payments keep using their original gateway even if the global
// setting changes later. A gateway that reports itself unavailable (e.g.
// Midtrans disabled) is skipped in favour of the manual fallback.
func (r *Registry) ResolveGateway(ctx context.Context, settings SettingsRepository, planCode string, planResolver PlanGatewayResolver) (Gateway, error) {
	// 1) Per-plan override.
	if planResolver != nil && planCode != "" {
		if override, err := planResolver(ctx, planCode); err == nil && override != "" {
			if g, ok := r.Get(override); ok && available(ctx, g) {
				return g, nil
			}
		}
	}
	// 2) Global default setting.
	if settings != nil {
		if name, err := settings.GetString(ctx, "payment_gateway_default"); err == nil && name != "" {
			if g, ok := r.Get(name); ok && available(ctx, g) {
				return g, nil
			}
		}
	}
	// 3) Emergency fallback.
	if g, ok := r.Get("manual"); ok {
		return g, nil
	}
	return nil, fmt.Errorf("%w: no gateway available", ErrGatewayNotConfigured)
}
