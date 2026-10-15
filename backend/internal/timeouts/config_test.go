package timeouts

import (
	"context"
	"testing"
	"time"

	"github.com/fintrack/backend/internal/domain"
)

// mapReader is a ConfigReader backed by a literal map (keys are setting keys).
type mapReader map[string]int

func (m mapReader) Int(_ context.Context, settingKey, _ string, def int) int {
	if v, ok := m[settingKey]; ok {
		return v
	}
	return def
}

func TestLoad_NilReaderYieldsNoOverrides(t *testing.T) {
	cfg := Load(context.Background(), nil)
	if cfg.For(domain.UseCaseVision) != 0 {
		t.Fatalf("expected zero override with nil reader, got %v", cfg.For(domain.UseCaseVision))
	}
	if len(cfg.Map()) != 0 {
		t.Fatalf("expected empty map with nil reader, got %v", cfg.Map())
	}
}

func TestLoad_ResolvesEachUseCase(t *testing.T) {
	r := mapReader{
		KeyChat:      60000,
		KeyVision:    120000,
		KeyVoice:     90000,
		KeyEmbedding: 45000,
		KeyReasoning: 180000,
	}
	cfg := Load(context.Background(), r)

	cases := []struct {
		uc   domain.UseCase
		want time.Duration
	}{
		{domain.UseCaseChat, 60 * time.Second},
		{domain.UseCaseVision, 120 * time.Second},
		{domain.UseCaseVoice, 90 * time.Second},
		{domain.UseCaseEmbedding, 45 * time.Second},
		{domain.UseCaseReasoning, 180 * time.Second},
	}
	for _, tc := range cases {
		if got := cfg.For(tc.uc); got != tc.want {
			t.Fatalf("For(%s) = %v, want %v", tc.uc, got, tc.want)
		}
	}

	// Map only carries configured (non-zero) use cases.
	m := cfg.Map()
	if len(m) != 5 {
		t.Fatalf("expected 5 overrides, got %d: %v", len(m), m)
	}
	if m[domain.UseCaseVision] != 120*time.Second {
		t.Fatalf("expected vision override in map, got %v", m[domain.UseCaseVision])
	}
}

func TestLoad_ZeroMeansNoOverride(t *testing.T) {
	// A use case left at 0 must not appear in the map, so the gateway falls back
	// to the per-route timeout_ms.
	r := mapReader{KeyVision: 120000}
	cfg := Load(context.Background(), r)
	m := cfg.Map()
	if _, ok := m[domain.UseCaseChat]; ok {
		t.Fatalf("chat should have no override, got %v", m[domain.UseCaseChat])
	}
	if m[domain.UseCaseVision] != 120*time.Second {
		t.Fatalf("expected vision override, got %v", m[domain.UseCaseVision])
	}
}

func TestLoad_ClampsOutOfRange(t *testing.T) {
	// Below the floor is clamped up to MinBudgetMS; above the ceiling down to
	// MaxBudgetMS. A negative value falls back (no override).
	r := mapReader{
		KeyChat:   1,          // -> clamped to 1s
		KeyVision: 99_999_999, // -> clamped to the proxy ceiling
		KeyVoice:  -5,         // -> no override
	}
	cfg := Load(context.Background(), r)
	if cfg.Chat != ms(MinBudgetMS) {
		t.Fatalf("expected chat clamped to min, got %v", cfg.Chat)
	}
	// An absurd value is clamped first to MaxBudgetMS and then to the proxy
	// ceiling (the outer layer that would sever the request anyway).
	if cfg.Vision != ms(ProxyCeilingMS()) {
		t.Fatalf("expected vision clamped to proxy ceiling, got %v", cfg.Vision)
	}
	if cfg.Voice != 0 {
		t.Fatalf("expected negative voice to fall back to 0, got %v", cfg.Voice)
	}
}

func TestLoad_ClientBudgetAutoDerived(t *testing.T) {
	// With no explicit client budget, it must exceed the slowest use case so the
	// browser never aborts before the server would.
	r := mapReader{KeyChat: 60000, KeyVision: 120000}
	cfg := Load(context.Background(), r)
	slowest := cfg.slowest()
	if slowest != 120*time.Second {
		t.Fatalf("expected slowest = vision, got %v", slowest)
	}
	if cfg.Client <= slowest {
		t.Fatalf("expected auto client budget > slowest (%v), got %v", slowest, cfg.Client)
	}
	if cfg.Client != slowest+ms(clientMarginMS) {
		t.Fatalf("expected slowest+margin, got %v", cfg.Client)
	}
}

func TestLoad_ExplicitClientBudgetWins(t *testing.T) {
	// 200s is above the default proxy ceiling (180s), so it is clamped to the
	// ceiling — a longer budget would be silently defeated by the proxy.
	r := mapReader{KeyVision: 120000, KeyClient: 200000}
	cfg := Load(context.Background(), r)
	if cfg.Client != ms(ProxyCeilingMS()) {
		t.Fatalf("expected client clamped to proxy ceiling, got %v", cfg.Client)
	}

	// Below the ceiling, the explicit value is honoured as-is.
	r2 := mapReader{KeyVision: 120000, KeyClient: 170000}
	cfg2 := Load(context.Background(), r2)
	if cfg2.Client != 170*time.Second {
		t.Fatalf("expected explicit client budget 170s, got %v", cfg2.Client)
	}
}

// TestHTTPDeadline_Ordering locks the layering invariant: the gateway's slowest
// per-call budget < HTTP write deadline < client budget. This is what guarantees
// the user sees the server's own error rather than a client-side abort.
func TestHTTPDeadline_Ordering(t *testing.T) {
	r := mapReader{KeyChat: 60000, KeyVision: 120000}
	cfg := Load(context.Background(), r)

	slowest := cfg.slowest()
	http := cfg.HTTPDeadline()

	if http <= slowest {
		t.Fatalf("http deadline %v must exceed slowest gateway budget %v", http, slowest)
	}
	if http >= cfg.Client {
		t.Fatalf("http deadline %v must be below client budget %v", http, cfg.Client)
	}
	if http != slowest+ms(httpDeadlineMarginMS) {
		t.Fatalf("expected slowest+margin, got %v", http)
	}
}

// TestHTTPDeadline_NoOverrideIsUnbounded verifies that with no use-case override
// the deadline is left to the server default rather than invented.
func TestHTTPDeadline_NoOverrideIsUnbounded(t *testing.T) {
	cfg := Load(context.Background(), mapReader{})
	if d := cfg.HTTPDeadline(); d != 0 {
		t.Fatalf("expected 0 (unbounded) without overrides, got %v", d)
	}
}

// TestHTTPDeadline_ClampedByExplicitClient verifies a small explicit client
// budget caps the HTTP deadline so it never outlives the browser.
func TestHTTPDeadline_ClampedByExplicitClient(t *testing.T) {
	// Vision 120s but the client only waits 125s: the HTTP deadline must not
	// exceed the client budget.
	r := mapReader{KeyVision: 120000, KeyClient: 125000}
	cfg := Load(context.Background(), r)
	if cfg.HTTPDeadline() > cfg.Client {
		t.Fatalf("http deadline %v must not exceed client %v", cfg.HTTPDeadline(), cfg.Client)
	}
}

// TestLoad_ClientBelowSlowestIsRaised locks the invariant guard: a client budget
// configured below the slowest use case (a misconfiguration) must be raised, so
// the browser never aborts while the server is still working.
func TestLoad_ClientBelowSlowestIsRaised(t *testing.T) {
	// Operator sets vision=120s but client=30s — the old code left this as-is,
	// breaking the gateway < HTTP < client ordering.
	r := mapReader{KeyVision: 120000, KeyClient: 30000}
	cfg := Load(context.Background(), r)

	slowest := cfg.slowest()
	if cfg.Client <= slowest {
		t.Fatalf("client %v must be raised above slowest %v", cfg.Client, slowest)
	}
	http := cfg.HTTPDeadline()
	if !(slowest < http && http < cfg.Client) {
		t.Fatalf("invariant violated: want slowest(%v) < http(%v) < client(%v)", slowest, http, cfg.Client)
	}
}

// TestHTTPDeadline_ExtremeCeilingIsUnbounded verifies that when both the use case
// and the client budget are pinned at the 30-minute ceiling there is no safe
// window, so the deadline is left unbounded instead of cutting the call short.
func TestHTTPDeadline_ExtremeCeilingIsUnbounded(t *testing.T) {
	r := mapReader{KeyVision: MaxBudgetMS, KeyClient: MaxBudgetMS}
	cfg := Load(context.Background(), r)
	if d := cfg.HTTPDeadline(); d != 0 {
		t.Fatalf("expected 0 (unbounded) at the ceiling, got %v", d)
	}
}

// TestLoad_ClampedToProxyCeiling verifies a budget above the Next.js proxy
// ceiling is clamped, so the UI can never persist a value the proxy would
// silently defeat.
func TestLoad_ClampedToProxyCeiling(t *testing.T) {
	t.Setenv(EnvProxyCeiling, "180000")
	// Vision well above the proxy ceiling (30 min) must come back at the ceiling.
	r := mapReader{KeyVision: 1_800_000, KeyClient: 1_800_000}
	cfg := Load(context.Background(), r)

	if cfg.Vision != 180*time.Second {
		t.Fatalf("expected vision clamped to proxy ceiling 180s, got %v", cfg.Vision)
	}
	if cfg.Client > 180*time.Second {
		t.Fatalf("expected client clamped to proxy ceiling, got %v", cfg.Client)
	}
}

// TestProxyCeilingMS_Default verifies the fallback matches next.config.ts.
func TestProxyCeilingMS_Default(t *testing.T) {
	t.Setenv(EnvProxyCeiling, "")
	if got := ProxyCeilingMS(); got != DefaultProxyCeilingMS {
		t.Fatalf("expected default %d, got %d", DefaultProxyCeilingMS, got)
	}
}
