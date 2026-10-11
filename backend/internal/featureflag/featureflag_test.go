package featureflag

import (
	"context"
	"testing"

	"github.com/google/uuid"

	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
)

type memRepo struct {
	flags map[string]*Flag
	err   error
	hits  int
}

func (m *memRepo) GetByKey(_ context.Context, key string) (*Flag, error) {
	m.hits++
	if m.err != nil {
		return nil, m.err
	}
	return m.flags[key], nil
}

func (m *memRepo) List(context.Context) ([]Flag, error) {
	var out []Flag
	for _, f := range m.flags {
		out = append(out, *f)
	}
	return out, nil
}

func (m *memRepo) Upsert(_ context.Context, f Flag, _ *uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	cp := f
	m.flags[f.Key] = &cp
	return nil
}

// TestIsEnabled_MissingFlagIsFalse asserts fail-closed on unknown flags.
func TestIsEnabled_MissingFlagIsFalse(t *testing.T) {
	svc := NewService(&memRepo{flags: map[string]*Flag{}}, nil)
	if svc.IsEnabled(context.Background(), "nope", nil) {
		t.Fatal("missing flag must be false")
	}
}

// TestIsEnabled_DisabledIsFalse asserts a disabled flag is off.
func TestIsEnabled_DisabledIsFalse(t *testing.T) {
	repo := &memRepo{flags: map[string]*Flag{"f": {Key: "f", IsEnabled: false, Rollout: 100}}}
	svc := NewService(repo, nil)
	if svc.IsEnabled(context.Background(), "f", new(uuid.New())) {
		t.Fatal("disabled flag must be false")
	}
}

// TestIsEnabled_FullRolloutIsTrue asserts enabled + 100% is on for everyone.
func TestIsEnabled_FullRolloutIsTrue(t *testing.T) {
	repo := &memRepo{flags: map[string]*Flag{"f": {Key: "f", IsEnabled: true, Rollout: 100}}}
	svc := NewService(repo, nil)
	if !svc.IsEnabled(context.Background(), "f", new(uuid.New())) {
		t.Fatal("enabled + 100% must be true")
	}
	// Even without a user (100% means everybody).
	if !svc.IsEnabled(context.Background(), "f", nil) {
		t.Fatal("enabled + 100% must be true for nil user too")
	}
}

// TestIsEnabled_ZeroRolloutIsFalse asserts 0% rollout is off.
func TestIsEnabled_ZeroRolloutIsFalse(t *testing.T) {
	repo := &memRepo{flags: map[string]*Flag{"f": {Key: "f", IsEnabled: true, Rollout: 0}}}
	svc := NewService(repo, nil)
	if svc.IsEnabled(context.Background(), "f", new(uuid.New())) {
		t.Fatal("0% rollout must be false")
	}
}

// TestIsEnabled_DeterministicBucketing asserts the same user always gets the
// same decision and that partial rollout splits the population.
func TestIsEnabled_DeterministicBucketing(t *testing.T) {
	repo := &memRepo{flags: map[string]*Flag{"f": {Key: "f", IsEnabled: true, Rollout: 50}}}
	svc := NewService(repo, nil)

	u := uuid.New()
	first := svc.IsEnabled(context.Background(), "f", new(u))
	for range 5 {
		if got := svc.IsEnabled(context.Background(), "f", new(u)); got != first {
			t.Fatal("evaluation must be deterministic for the same user")
		}
	}

	// With enough users, ~half should be enabled (sanity bound 20%..80%).
	enabled := 0
	const n = 500
	for range n {
		if svc.IsEnabled(context.Background(), "f", new(uuid.New())) {
			enabled++
		}
	}
	if enabled < n/5 || enabled > n*4/5 {
		t.Fatalf("50%% rollout produced %d/%d enabled (expected roughly half)", enabled, n)
	}
}

// TestIsEnabled_NilUserPartialRollout asserts nil user cannot be bucketed.
func TestIsEnabled_NilUserPartialRollout(t *testing.T) {
	repo := &memRepo{flags: map[string]*Flag{"f": {Key: "f", IsEnabled: true, Rollout: 50}}}
	svc := NewService(repo, nil)
	if svc.IsEnabled(context.Background(), "f", nil) {
		t.Fatal("nil user with partial rollout must be false")
	}
}

// TestIsEnabled_RepoErrorIsFailClosed asserts errors yield false.
func TestIsEnabled_RepoErrorIsFailClosed(t *testing.T) {
	repo := &memRepo{err: context.DeadlineExceeded}
	svc := NewService(repo, nil)
	if svc.IsEnabled(context.Background(), "f", new(uuid.New())) {
		t.Fatal("repository error must fail closed")
	}
}

// TestIsEnabled_Caching asserts a second call is served from cache (repo hit
// count stays at 1).
func TestIsEnabled_Caching(t *testing.T) {
	repo := &memRepo{flags: map[string]*Flag{"f": {Key: "f", IsEnabled: true, Rollout: 100}}}
	cache := infraRedis.NewCacheEngine(nil) // in-memory backend
	svc := NewService(repo, cache)

	svc.IsEnabled(context.Background(), "f", new(uuid.New()))
	svc.IsEnabled(context.Background(), "f", new(uuid.New()))

	if repo.hits != 1 {
		t.Fatalf("expected 1 repository hit (second served from cache), got %d", repo.hits)
	}
}

// TestIsEnabledForPlatform_IgnoresRollout is the regression guard for the bug
// that left the centralized AI gateway permanently disabled: the flag was
// enabled with rollout=25, and IsEnabled(…, nil) returns false for partial
// rollout, so the gateway never turned on.
func TestIsEnabledForPlatform_IgnoresRollout(t *testing.T) {
	repo := &memRepo{flags: map[string]*Flag{
		"gateway": {Key: "gateway", IsEnabled: true, Rollout: 25},
	}}
	svc := NewService(repo, nil)

	if !svc.IsEnabledForPlatform(context.Background(), "gateway") {
		t.Fatal("enabled flag with partial rollout must read as ON for the platform")
	}
}

func TestIsEnabledForPlatform_FailClosed(t *testing.T) {
	cases := map[string]*Flag{
		"missing":  nil,
		"disabled": {Key: "disabled", IsEnabled: false, Rollout: 100},
		"zeroRoll": {Key: "zeroRoll", IsEnabled: true, Rollout: 0},
	}
	for name, flag := range cases {
		repo := &memRepo{flags: map[string]*Flag{}}
		if flag != nil {
			repo.flags[flag.Key] = flag
		}
		svc := NewService(repo, nil)
		if svc.IsEnabledForPlatform(context.Background(), name) {
			t.Fatalf("%s: must be false (fail-closed)", name)
		}
	}

	// Repository error must also be false.
	svc := NewService(&memRepo{flags: map[string]*Flag{}, err: context.DeadlineExceeded}, nil)
	if svc.IsEnabledForPlatform(context.Background(), "anything") {
		t.Fatal("repository error must be false")
	}
}

// TestIsEnabledForPlatform_FullRollout is the ordinary ON case.
func TestIsEnabledForPlatform_FullRollout(t *testing.T) {
	repo := &memRepo{flags: map[string]*Flag{
		"gateway": {Key: "gateway", IsEnabled: true, Rollout: 100},
	}}
	svc := NewService(repo, nil)
	if !svc.IsEnabledForPlatform(context.Background(), "gateway") {
		t.Fatal("fully rolled-out enabled flag must be ON")
	}
}

var _ = infraRedis.NewCacheEngine
