// Package timeouts resolves the runtime timeout budget for AI calls.
//
// Every value is dynamic: an explicit `settings` row wins, otherwise the
// process environment supplies the default (see runtimeconfig.Resolver), and
// when neither is set a documented fallback applies. A Super Admin can retune
// the budgets from the UI and the change applies to in-flight routing within
// seconds — no redeploy, no restart.
//
// The package deliberately knows nothing about the gateway or HTTP layers: it
// only turns the settings keys into durations, so the gateway, the HTTP
// middleware and the client-config endpoint all read one source of truth.
package timeouts

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fintrack/backend/internal/domain"
)

// Setting keys (must match the writableSettings allowlist in internal/admin).
const (
	KeyChat      = "ai_timeout_chat_ms"
	KeyVision    = "ai_timeout_vision_ms"
	KeyVoice     = "ai_timeout_voice_ms"
	KeyEmbedding = "ai_timeout_embedding_ms"
	KeyReasoning = "ai_timeout_reasoning_ms"
	// KeyClient is the budget the browser should allow for an AI request before
	// aborting. It must be >= the slowest server-side use case, otherwise the
	// client gives up while the server is still working.
	KeyClient = "ai_timeout_client_ms"
)

// Env keys, used only when no settings row exists.
const (
	EnvChat      = "AI_TIMEOUT_CHAT_MS"
	EnvVision    = "AI_TIMEOUT_VISION_MS"
	EnvVoice     = "AI_TIMEOUT_VOICE_MS"
	EnvEmbedding = "AI_TIMEOUT_EMBEDDING_MS"
	EnvReasoning = "AI_TIMEOUT_REASONING_MS"
	EnvClient    = "AI_TIMEOUT_CLIENT_MS"
	// EnvProxyCeiling is the same variable the Next.js rewrite proxy reads
	// (`experimental.proxyTimeout`). It is the hard outer ceiling: a request that
	// outlives it is severed by the proxy no matter what the AI budget says, so
	// the budgets must never exceed it.
	EnvProxyCeiling = "API_PROXY_TIMEOUT_MS"
)

// DefaultProxyCeilingMS mirrors next.config.ts's default (180s).
const DefaultProxyCeilingMS = 180_000

// ProxyCeilingMS returns the outer proxy ceiling in milliseconds. Every AI
// budget is clamped to it: configuring a longer budget would be silently
// defeated by the proxy, which is exactly the confusing "it still times out"
// symptom this feature exists to remove.
func ProxyCeilingMS() int {
	v := strings.TrimSpace(os.Getenv(EnvProxyCeiling))
	if n, err := strconv.Atoi(v); err == nil && n >= MinBudgetMS {
		return n
	}
	return DefaultProxyCeilingMS
}

// Bounds applied to every resolved value so a typo cannot disable the safety
// net (too small) or let a request hang forever (too large).
const (
	MinBudgetMS = 1_000       // 1s
	MaxBudgetMS = 30 * 60_000 // 30 min
	// clientMarginMS is added to the slowest use case when the client budget is
	// left at "auto" (0), so the browser waits slightly longer than the server.
	clientMarginMS = 15_000
	// httpDeadlineMarginMS is added to the slowest use case to derive the HTTP
	// write deadline. It is smaller than clientMarginMS so the ordering is
	// gateway < HTTP < client: the handler can still flush its error after the
	// upstream call times out, and the browser still waits past that.
	httpDeadlineMarginMS = 10_000
)

// ConfigReader resolves one setting with DB-over-env precedence. Satisfied by
// *runtimeconfig.Resolver; kept as an interface so this package is testable
// without a database.
type ConfigReader interface {
	Int(ctx context.Context, settingKey, envKey string, def int) int
}

// Config is the effective timeout budget. A zero duration for a use case means
// "no override": the caller falls back to the per-route timeout_ms and then to
// the gateway default.
type Config struct {
	Chat      time.Duration
	Vision    time.Duration
	Voice     time.Duration
	Embedding time.Duration
	Reasoning time.Duration
	// Client is the budget handed to the browser for AI requests.
	Client time.Duration
}

// Load resolves the effective configuration. A nil reader yields zero overrides
// (the gateway keeps its own defaults), so callers never have to guard.
func Load(ctx context.Context, r ConfigReader) Config {
	if r == nil {
		return Config{}
	}
	cfg := Config{
		Chat:      ms(r.Int(ctx, KeyChat, EnvChat, 0)),
		Vision:    ms(r.Int(ctx, KeyVision, EnvVision, 0)),
		Voice:     ms(r.Int(ctx, KeyVoice, EnvVoice, 0)),
		Embedding: ms(r.Int(ctx, KeyEmbedding, EnvEmbedding, 0)),
		Reasoning: ms(r.Int(ctx, KeyReasoning, EnvReasoning, 0)),
		Client:    ms(r.Int(ctx, KeyClient, EnvClient, 0)),
	}
	return cfg.normalize()
}

// For returns the override for a use case (0 when none is configured).
func (c Config) For(uc domain.UseCase) time.Duration {
	switch uc {
	case domain.UseCaseChat:
		return c.Chat
	case domain.UseCaseVision:
		return c.Vision
	case domain.UseCaseVoice:
		return c.Voice
	case domain.UseCaseEmbedding:
		return c.Embedding
	case domain.UseCaseReasoning:
		return c.Reasoning
	default:
		return 0
	}
}

// HTTPDeadline is how long the connection may stay open while the handler waits
// for the upstream call. It sits strictly between the gateway's per-call budget
// and the client's budget:
//
//	gateway (slowest use case)  <  HTTPDeadline  <  Client
//
// so that when the upstream call times out, the handler still has room to write
// its friendly error before either the socket closes or the browser gives up.
//
// normalize() guarantees Client >= slowest+clientMargin, so the window is never
// empty in practice. The guard below still defends the extreme case (both
// clamped to the 30-minute ceiling) by returning 0 — "unbounded" — rather than
// a deadline that would cut the upstream call short.
func (c Config) HTTPDeadline() time.Duration {
	slowest := c.slowest()
	if slowest <= 0 {
		// No use-case override: leave the deadline to the server default (0 =
		// "clear", i.e. unbounded) rather than inventing a budget.
		return 0
	}
	d := slowest + ms(httpDeadlineMarginMS)
	if c.Client > 0 && d > c.Client {
		// No headroom between the upstream budget and the client budget: prefer
		// an unbounded socket over one that severs a still-running call.
		if c.Client <= slowest {
			return 0
		}
		d = c.Client
	}
	return d
}

// Map returns the overrides keyed by use case, for the gateway to install.
func (c Config) Map() map[domain.UseCase]time.Duration {
	out := make(map[domain.UseCase]time.Duration, len(domain.AllUseCases))
	for _, uc := range domain.AllUseCases {
		if d := c.For(uc); d > 0 {
			out[uc] = d
		}
	}
	return out
}

// normalize clamps each configured value into the safe range and resolves the
// client budget so the layering invariant always holds:
//
//	gateway (slowest use case)  <  HTTPDeadline  <  Client
//
// When the operator leaves the client budget unset (0) it is derived from the
// slowest use case plus a margin. When it IS set but falls below that floor —
// a misconfiguration that would make the browser abort while the server is
// still working, so the user sees a generic timeout instead of the server's own
// friendly error — it is raised to the floor. The raise is deliberate: a client
// budget shorter than the server's own worst case is never useful.
func (c Config) normalize() Config {
	c.Chat = clamp(c.Chat)
	c.Vision = clamp(c.Vision)
	c.Voice = clamp(c.Voice)
	c.Embedding = clamp(c.Embedding)
	c.Reasoning = clamp(c.Reasoning)

	// The Next.js rewrite proxy is the outermost layer and severs anything that
	// outlives it, so a budget above its ceiling can never be honoured. Clamping
	// here (instead of trusting the operator) turns a silently-ignored value into
	// an effective one, and keeps the documented ordering true end-to-end.
	ceiling := ms(ProxyCeilingMS())
	c.Chat = minDur(c.Chat, ceiling)
	c.Vision = minDur(c.Vision, ceiling)
	c.Voice = minDur(c.Voice, ceiling)
	c.Embedding = minDur(c.Embedding, ceiling)
	c.Reasoning = minDur(c.Reasoning, ceiling)

	slowest := c.slowest()
	floor := slowest + ms(clientMarginMS)
	if c.Client <= 0 {
		if slowest > 0 {
			c.Client = floor
		}
	} else if slowest > 0 && c.Client < floor {
		c.Client = floor
	}
	c.Client = clamp(c.Client)
	// The client budget can never exceed the proxy ceiling either, or the browser
	// would wait for a reply the proxy has already abandoned.
	c.Client = minDur(c.Client, ceiling)

	// If clamping the ceiling squeezed the client budget back down to (or below)
	// the slowest use case, there is no room for a safe HTTP deadline; fall back
	// to leaving the socket unbounded rather than cutting it short. This only
	// triggers at the extreme ceiling.
	return c
}

// minDur returns the smaller of d and cap, treating 0 as "no value" (so it does
// not accidentally clamp a real duration down to zero).
func minDur(d, cap time.Duration) time.Duration {
	if d <= 0 {
		return d
	}
	if cap > 0 && d > cap {
		return cap
	}
	return d
}

// slowest returns the largest configured use-case override.
func (c Config) slowest() time.Duration {
	var max time.Duration
	for _, d := range []time.Duration{c.Chat, c.Vision, c.Voice, c.Embedding, c.Reasoning} {
		if d > max {
			max = d
		}
	}
	return max
}

// clamp zeroes out-of-range values (so they fall back) and caps absurd ones.
func clamp(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	lo := ms(MinBudgetMS)
	hi := ms(MaxBudgetMS)
	if d < lo {
		return lo
	}
	if d > hi {
		return hi
	}
	return d
}

// ms converts a millisecond count into a Duration.
func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }
