// Package aituning applies the AI runtime knobs that live in the `settings`
// table to the in-memory components that hold them (the gateway's per-use-case
// timeouts and the HTTP layer's write-deadline budget).
//
// It exists so there is exactly one place that reads the keys and pushes them
// into the running process, shared by:
//
//   - boot (apply once before serving),
//   - the Super Admin settings update callback (apply immediately on the pod
//     that served the request), and
//   - the Pub/Sub watcher (apply on every other pod).
//
// Because the applier re-reads the database rather than trusting a broadcast
// payload, a missed message self-heals on the next change, and a partial
// rollout converges.
package aituning

import (
	"context"

	"github.com/fintrack/backend/internal/aigateway/gateway"
	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/timeouts"
	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

// Reader resolves one setting with DB-over-env precedence (satisfied by
// *runtimeconfig.Resolver).
type Reader interface {
	timeouts.ConfigReader
}

// Applier pushes the resolved AI timeout budget into the gateway and the HTTP
// middleware. Both sinks are optional (nil in tests / partial wiring).
type Applier struct {
	gateway *gateway.Service
	reader  Reader
}

// New builds an Applier. gateway may be nil; reader may be nil (defaults apply).
func New(gw *gateway.Service, r Reader) *Applier {
	return &Applier{gateway: gw, reader: r}
}

// Apply resolves the current budget and installs it everywhere. It is safe to
// call concurrently and while requests are in flight: both sinks swap their
// state atomically, so there is no window where a request sees a torn value.
func (a *Applier) Apply(ctx context.Context) {
	if a == nil {
		return
	}
	cfg := timeouts.Load(ctx, a.reader)

	// The gateway owns the per-upstream-call budget; a use case without an
	// override keeps its route's timeout_ms (then the gateway default).
	if a.gateway != nil {
		a.gateway.SetUseCaseTimeouts(cfg.Map())
	}

	// The HTTP layer owns how long the connection may stay open while the
	// handler waits. It is derived to sit between the gateway's per-call budget
	// and the client's budget, so the handler can still flush its response after
	// an upstream timeout (see timeouts.Config.HTTPDeadline).
	middleware.SetAIWriteTimeout(cfg.HTTPDeadline())

	logger.Debug("ai timeout budget applied",
		zap.Duration("chat", cfg.Chat),
		zap.Duration("vision", cfg.Vision),
		zap.Duration("voice", cfg.Voice),
		zap.Duration("embedding", cfg.Embedding),
		zap.Duration("reasoning", cfg.Reasoning),
		zap.Duration("http_deadline", cfg.HTTPDeadline()),
		zap.Duration("client", cfg.Client))
}

// Load exposes the resolved config to callers that need to report it (e.g. the
// client-config endpoint) without duplicating the precedence rules.
func Load(ctx context.Context, r Reader) timeouts.Config {
	return timeouts.Load(ctx, r)
}
