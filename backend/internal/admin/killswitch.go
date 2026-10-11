package admin

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
)

// KillSwitchKey is the settings row holding the AI emergency-stop state.
const KillSwitchKey = "ai_kill_switch"

// KillSwitchChannel is the Redis Pub/Sub channel used to broadcast kill-switch
// changes to every backend instance (< 100ms propagation, P06-016).
const KillSwitchChannel = "fintrack:ai-gateway:control"

// KillSwitchStore holds the AI emergency-stop matrix: a global flag plus a
// per-use-case override. Reads are lock-free snapshots so the gateway hot path
// never blocks; writes persist to `settings` and fan out over Pub/Sub.
type KillSwitchStore struct {
	pool   *pgxpool.Pool
	pubsub *infraRedis.PubSub

	mu    sync.RWMutex
	state map[string]bool
}

// NewKillSwitchStore builds an empty store. Call Load then Watch at startup.
func NewKillSwitchStore(pool *pgxpool.Pool, pubsub *infraRedis.PubSub) *KillSwitchStore {
	return &KillSwitchStore{pool: pool, pubsub: pubsub, state: map[string]bool{}}
}

// Enabled reports whether a use case is stopped (global flag or per-target).
func (k *KillSwitchStore) Enabled(useCase string) bool {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.state["global"] || k.state[useCase]
}

// Snapshot returns a copy of the current matrix.
func (k *KillSwitchStore) Snapshot() map[string]bool {
	k.mu.RLock()
	defer k.mu.RUnlock()
	out := make(map[string]bool, len(k.state))
	for k2, v := range k.state {
		out[k2] = v
	}
	return out
}

// Matrix returns every canonical target ("global" plus each use case) with the
// persisted switch state applied; targets never toggled read as false. The UI
// relies on all keys being present so toggles are not rendered as unknown.
func (k *KillSwitchStore) Matrix() map[string]bool {
	out := k.Snapshot()
	for t := range killSwitchTargets() {
		if _, ok := out[t]; !ok {
			out[t] = false
		}
	}
	return out
}

// killSwitchTargets is the canonical target set exposed to operators.
func killSwitchTargets() map[string]bool {
	out := make(map[string]bool, len(domain.AllUseCases)+1)
	out["global"] = true
	for _, uc := range domain.AllUseCases {
		out[string(uc)] = true
	}
	return out
}

// Load reads the persisted state from the settings table.
func (k *KillSwitchStore) Load(ctx context.Context) error {
	var raw []byte
	err := k.pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = $1`, KillSwitchKey).Scan(&raw)
	if err != nil {
		return err
	}
	var m map[string]bool
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	k.mu.Lock()
	k.state = m
	k.mu.Unlock()
	return nil
}

// apply merges a payload into local state (used on Pub/Sub delivery).
func (k *KillSwitchStore) apply(payload string) {
	var m map[string]bool
	if json.Unmarshal([]byte(payload), &m) != nil {
		return
	}
	k.mu.Lock()
	k.state = m
	k.mu.Unlock()
}

// Set toggles a target ("global" or a use case), persists it, updates local
// state, and broadcasts the new matrix to all instances.
func (k *KillSwitchStore) Set(ctx context.Context, target string, enabled bool) error {
	k.mu.Lock()
	next := make(map[string]bool, len(k.state))
	for kk, v := range k.state {
		next[kk] = v
	}
	next[target] = enabled
	k.state = next
	k.mu.Unlock()

	payload, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if _, err := k.pool.Exec(ctx, `
		INSERT INTO settings (key, value, description)
		VALUES ($1, $2, 'Status Emergency Kill Switch AI Gateway per modalitas')
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`,
		KillSwitchKey, payload); err != nil {
		return err
	}
	if k.pubsub != nil && k.pubsub.Enabled() {
		_ = k.pubsub.Publish(ctx, KillSwitchChannel, string(payload))
	}
	return nil
}

// Watch subscribes to kill-switch broadcasts and keeps local state fresh.
// Returns a stop function; a no-op when Pub/Sub is disabled (single instance).
func (k *KillSwitchStore) Watch(ctx context.Context) func() {
	if k.pubsub == nil || !k.pubsub.Enabled() {
		return func() {}
	}
	return k.pubsub.Subscribe(ctx, KillSwitchChannel, k.apply)
}
