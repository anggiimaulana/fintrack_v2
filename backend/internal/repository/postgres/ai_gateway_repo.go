package postgres

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/infrastructure/crypto"
)

// AIGatewayRepo reads pooled AI endpoints and per-use-case routes. Active
// routes are cached in-memory with a short TTL so the hot path avoids a DB
// round-trip; the cache is invalidated on admin mutations.
type AIGatewayRepo struct {
	pool *pgxpool.Pool

	mu      sync.RWMutex
	cache   map[domain.UseCase][]*domain.AIRoute
	expires time.Time
	ttl     time.Duration
}

// NewAIGatewayRepo builds the repository with a 60s route cache.
func NewAIGatewayRepo(pool *pgxpool.Pool) *AIGatewayRepo {
	return &AIGatewayRepo{
		pool:  pool,
		cache: make(map[domain.UseCase][]*domain.AIRoute),
		ttl:   60 * time.Second,
	}
}

// GetActiveRoutesByUseCase returns active routes ordered by priority ASC,
// each with its endpoint joined. Served from the in-memory cache when fresh.
func (r *AIGatewayRepo) GetActiveRoutesByUseCase(ctx context.Context, useCase domain.UseCase) ([]*domain.AIRoute, error) {
	r.mu.RLock()
	if time.Now().Before(r.expires) {
		if routes, ok := r.cache[useCase]; ok {
			r.mu.RUnlock()
			return routes, nil
		}
	}
	r.mu.RUnlock()

	routes, err := r.loadRoutes(ctx, useCase)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	if time.Now().After(r.expires) {
		// Cache expired: start a new window and drop stale entries.
		r.cache = make(map[domain.UseCase][]*domain.AIRoute)
		r.expires = time.Now().Add(r.ttl)
	}
	r.cache[useCase] = routes
	r.mu.Unlock()

	return routes, nil
}

// HasActiveRoutes reports whether at least one route is configured for the use
// case. Reuses the cached route lookup so a readiness check is cheap.
func (r *AIGatewayRepo) HasActiveRoutes(ctx context.Context, useCase domain.UseCase) (bool, error) {
	routes, err := r.GetActiveRoutesByUseCase(ctx, useCase)
	if err != nil {
		return false, err
	}
	return len(routes) > 0, nil
}

// HasHealthyEndpoint reports whether any route for the use case points at an
// endpoint marked healthy. "unknown" counts as not healthy: a fresh endpoint
// that has never been probed is not evidence that the gateway can serve traffic.
func (r *AIGatewayRepo) HasHealthyEndpoint(ctx context.Context, useCase domain.UseCase) (bool, error) {
	routes, err := r.GetActiveRoutesByUseCase(ctx, useCase)
	if err != nil {
		return false, err
	}
	for _, rt := range routes {
		if rt.Endpoint != nil && rt.Endpoint.Health == "healthy" {
			return true, nil
		}
	}
	return false, nil
}

func (r *AIGatewayRepo) loadRoutes(ctx context.Context, useCase domain.UseCase) ([]*domain.AIRoute, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT rt.id, rt.use_case, rt.endpoint_id, rt.model, rt.combo, rt.priority,
               rt.is_active, rt.timeout_ms, rt.max_retries, rt.created_at, rt.updated_at,
               ep.id, ep.name, ep.base_url, ep.api_key_encrypted, ep.provider, ep.priority,
               ep.status, ep.health, ep.last_check_at, ep.last_error, ep.modalities, ep.weight,
               ep.created_at, ep.updated_at
        FROM ai_routes rt
        JOIN ai_endpoints ep ON ep.id = rt.endpoint_id
        WHERE rt.use_case = $1 AND rt.is_active = true
        ORDER BY rt.priority ASC, ep.priority ASC`, string(useCase))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.AIRoute
	for rows.Next() {
		rt := &domain.AIRoute{}
		ep := &domain.AIEndpoint{}
		if err := rows.Scan(
			&rt.ID, &rt.UseCase, &rt.EndpointID, &rt.Model, &rt.Combo, &rt.Priority,
			&rt.IsActive, &rt.TimeoutMS, &rt.MaxRetries, &rt.CreatedAt, &rt.UpdatedAt,
			&ep.ID, &ep.Name, &ep.BaseURL, &ep.APIKeyEncrypted, &ep.Provider, &ep.Priority,
			&ep.Status, &ep.Health, &ep.LastCheckAt, &ep.LastError, &ep.Modalities, &ep.Weight,
			&ep.CreatedAt, &ep.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rt.Endpoint = ep
		out = append(out, rt)
	}
	return out, rows.Err()
}

// GetByID loads a single endpoint.
func (r *AIGatewayRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.AIEndpoint, error) {
	ep := &domain.AIEndpoint{}
	err := r.pool.QueryRow(ctx, `
        SELECT id, name, base_url, api_key_encrypted, provider, priority, status, health,
               last_check_at, last_error, modalities, weight, created_at, updated_at
        FROM ai_endpoints WHERE id = $1`, id).
		Scan(&ep.ID, &ep.Name, &ep.BaseURL, &ep.APIKeyEncrypted, &ep.Provider, &ep.Priority,
			&ep.Status, &ep.Health, &ep.LastCheckAt, &ep.LastError, &ep.Modalities, &ep.Weight,
			&ep.CreatedAt, &ep.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ep, nil
}

// ListEndpoints returns all endpoints ordered by priority.
func (r *AIGatewayRepo) ListEndpoints(ctx context.Context) ([]*domain.AIEndpoint, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, name, base_url, api_key_encrypted, provider, priority, status, health,
               last_check_at, last_error, modalities, weight, created_at, updated_at
        FROM ai_endpoints ORDER BY priority ASC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.AIEndpoint
	for rows.Next() {
		ep := &domain.AIEndpoint{}
		if err := rows.Scan(&ep.ID, &ep.Name, &ep.BaseURL, &ep.APIKeyEncrypted, &ep.Provider,
			&ep.Priority, &ep.Status, &ep.Health, &ep.LastCheckAt, &ep.LastError, &ep.Modalities,
			&ep.Weight, &ep.CreatedAt, &ep.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, ep)
	}
	return out, rows.Err()
}

// UpdateEndpointHealth persists the latest health probe result.
func (r *AIGatewayRepo) UpdateEndpointHealth(ctx context.Context, id uuid.UUID, health string, lastCheck time.Time, lastErr *string) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE ai_endpoints SET health = $2, last_check_at = $3, last_error = $4, updated_at = now()
        WHERE id = $1`, id, health, lastCheck, lastErr)
	return err
}

// InvalidateCache drops the in-memory route cache (call after admin changes).
func (r *AIGatewayRepo) InvalidateCache() {
	r.mu.Lock()
	r.cache = make(map[domain.UseCase][]*domain.AIRoute)
	r.expires = time.Time{}
	r.mu.Unlock()
}

// DecryptKey decrypts an endpoint's stored API key using the master key.
// Returns an empty string for keyless endpoints (e.g. local 9router).
func DecryptKey(encrypted *string, masterKey []byte) (string, error) {
	if encrypted == nil || *encrypted == "" {
		return "", nil
	}
	plain, err := crypto.Decrypt(*encrypted, masterKey)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
