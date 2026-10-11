// Package featureflag evaluates runtime feature flags. Flags are stored in
// PostgreSQL and cached (Redis when available, otherwise in-memory) so they can
// be toggled without a redeploy. Evaluation is fail-closed: any error or missing
// flag yields false.
package featureflag

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"time"

	"github.com/google/uuid"

	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
)

// Flag mirrors a row of the feature_flags table.
type Flag struct {
	Key         string         `json:"key"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	IsEnabled   bool           `json:"is_enabled"`
	Rollout     int            `json:"rollout"`
	Rules       map[string]any `json:"rules"`
}

// Repository reads and writes flags in persistent storage.
type Repository interface {
	GetByKey(ctx context.Context, key string) (*Flag, error)
	List(ctx context.Context) ([]Flag, error)
	// Upsert creates the flag when absent, otherwise updates it.
	Upsert(ctx context.Context, f Flag, updatedBy *uuid.UUID) error
}

// Service evaluates flags with a cache in front of the repository.
type Service struct {
	repo  Repository
	cache *infraRedis.CacheEngine
	ttl   time.Duration
}

// NewService builds a flag service. Pass a CacheEngine (Redis or in-memory) for
// caching; pass nil to always hit the repository.
func NewService(repo Repository, cache *infraRedis.CacheEngine) *Service {
	return &Service{repo: repo, cache: cache, ttl: 30 * time.Second}
}

func cacheKeyFor(key string) string { return "ff:flag:" + key }

// get returns a flag from cache or the repository.
func (s *Service) get(ctx context.Context, key string) (*Flag, error) {
	if s.cache != nil {
		if raw, ok := s.cache.Get(ctx, cacheKeyFor(key)); ok {
			var f Flag
			if err := json.Unmarshal(raw, &f); err == nil {
				return &f, nil
			}
		}
	}

	f, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if f != nil && s.cache != nil {
		if raw, err := json.Marshal(f); err == nil {
			s.cache.Set(ctx, cacheKeyFor(key), raw, s.ttl)
		}
	}
	return f, nil
}

// List returns every flag (admin surface).
func (s *Service) List(ctx context.Context) ([]Flag, error) { return s.repo.List(ctx) }

// GetByKey returns a single flag (admin surface), nil when absent.
func (s *Service) GetByKey(ctx context.Context, key string) (*Flag, error) {
	return s.repo.GetByKey(ctx, key)
}

// Set upserts a flag and drops its cached value so the change is visible
// immediately across instances.
func (s *Service) Set(ctx context.Context, f Flag, updatedBy *uuid.UUID) error {
	if f.Rules == nil {
		f.Rules = map[string]any{}
	}
	if err := s.repo.Upsert(ctx, f, updatedBy); err != nil {
		return err
	}
	s.Invalidate(ctx, f.Key)
	return nil
}

// Invalidate drops the cached value for a flag (call after admin changes).
func (s *Service) Invalidate(ctx context.Context, key string) {
	if s.cache != nil {
		s.cache.DeletePattern(ctx, cacheKeyFor(key))
	}
}

// IsEnabled reports whether a flag is active for the given user.
//
// Rules:
//   - Missing flag or repository error => false (fail-closed).
//   - is_enabled=false => false.
//   - rollout<=0 => false; rollout>=100 => true.
//   - otherwise => deterministic bucket per (key,userID) < rollout.
//     A nil userID cannot be bucketed, so it returns false for partial rollout.
func (s *Service) IsEnabled(ctx context.Context, key string, userID *uuid.UUID) bool {
	f, err := s.get(ctx, key)
	if err != nil || f == nil {
		return false
	}
	if !f.IsEnabled {
		return false
	}
	if f.Rollout <= 0 {
		return false
	}
	if f.Rollout >= 100 {
		return true
	}
	if userID == nil {
		return false
	}
	return bucket(key, userID.String()) < f.Rollout
}

// IsEnabledForPlatform reports whether a flag is switched on for the whole
// platform, ignoring per-user rollout buckets.
//
// Use this for flags that are architectural switches rather than gradual
// rollouts — e.g. "route all AI traffic through the centralized gateway". For
// those, a partial `rollout` percentage is meaningless: the system cannot be in
// two modes at once, and reading them with IsEnabled(…, nil) silently returns
// false forever (which is exactly how the centralized AI gateway ended up
// permanently disabled in production).
//
// Rules: missing flag / repository error / is_enabled=false / rollout<=0 => false
// (fail-closed). Otherwise true, regardless of the rollout percentage.
func (s *Service) IsEnabledForPlatform(ctx context.Context, key string) bool {
	f, err := s.get(ctx, key)
	if err != nil || f == nil {
		return false
	}
	return f.IsEnabled && f.Rollout > 0
}

// bucket returns a deterministic 0..99 bucket for (key, subject).
func bucket(key, subject string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key + ":" + subject))
	return int(h.Sum32() % 100)
}
