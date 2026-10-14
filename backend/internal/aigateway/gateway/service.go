// Package gateway wires the pieces of the centralized AI gateway together: a
// per-use-case route pool (repository), a circuit breaker, an OpenAI-compatible
// client, and token metering. It exposes the domain.AIGatewayService interface
// used by business code.
package gateway

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/aigateway/breaker"
	"github.com/fintrack/backend/internal/aigateway/client"
	"github.com/fintrack/backend/internal/aigateway/metering"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/infrastructure/crypto"
	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

// Config configures the gateway service.
type Config struct {
	// EncryptionKey is the master key used to decrypt endpoint API keys.
	EncryptionKey []byte
	// DefaultTimeout is used when a route does not specify timeout_ms.
	DefaultTimeout time.Duration
	// Breaker config; defaults applied when zero-valued.
	Breaker breaker.Config
	// KillSwitch, when non-nil, reports whether a use case is administratively
	// disabled by the Super Admin emergency stop. Nil means all enabled.
	KillSwitch func(uc domain.UseCase) bool
	// OnRouteStateChange, when set, is called when a route is quarantined
	// (opened=true) or recovers (opened=false). Nil-safe: when unset, route
	// quarantine is silent (only logged). Used to emit admin realtime alerts.
	OnRouteStateChange func(ev RouteStateChange)
}

// RouteStateChange describes a route entering or leaving quarantine.
type RouteStateChange struct {
	UseCase    string
	Model      string
	Endpoint   string
	EndpointID string
	Combo      string
	// Opened is true when the route was quarantined, false when it recovered.
	Opened bool
	// Reason is the last failure that tripped the route, when known.
	Reason string
}

// ErrKillSwitch is returned when the emergency kill switch disables a use case.
var ErrKillSwitch = errors.New("ai gateway disabled by kill switch")

// Service implements domain.AIGatewayService over a pooled set of endpoints.
type Service struct {
	repo    domain.AIEndpointRepository
	breaker *breaker.Breaker
	cfg     Config

	// routeHealth quarantines individual ROUTES (endpoint+model), not whole
	// endpoints. This is the "smart" part of failover: a dead model is skipped
	// while its healthy siblings on the same endpoint keep serving.
	routeHealth *routeHealth

	rrMu    sync.Mutex
	rrState map[domain.UseCase]map[string]int64 // use_case -> combo -> currentWeight
}


// New builds a gateway service.
func New(repo domain.AIEndpointRepository, cfg Config) *Service {
	if cfg.DefaultTimeout <= 0 {
		cfg.DefaultTimeout = 30 * time.Second
	}
	bc := cfg.Breaker
	if bc.FailureThreshold == 0 && bc.Cooldown == 0 {
		bc = breaker.DefaultConfig()
	}
	s := &Service{
		repo:        repo,
		breaker:     breaker.New(bc),
		routeHealth: newRouteHealth(routeBreakerFailureThreshold, routeBreakerCooldown),
		cfg:         cfg,
		rrState:     make(map[domain.UseCase]map[string]int64),
	}
	s.seedWRR()
	s.withRouteCallbacks(cfg)
	return s
}

func (s *Service) seedWRR() {
	s.rrMu.Lock()
	defer s.rrMu.Unlock()
	// Pre-seed pseudo-random offsets for standard use cases to prevent multi-instance lockstep.
	for _, uc := range []domain.UseCase{
		domain.UseCaseChat,
		domain.UseCaseVision,
		domain.UseCaseVoice,
		domain.UseCaseEmbedding,
		domain.UseCaseReasoning,
	} {
		s.rrState[uc] = make(map[string]int64)
	}
}

// Breaker exposes the breaker (for health checks / diagnostics).
func (s *Service) Breaker() *breaker.Breaker { return s.breaker }

// SetBreakerConfig retunes the endpoint breaker at runtime (Super Admin settings
// change) without losing accumulated state.
func (s *Service) SetBreakerConfig(cfg breaker.Config) { s.breaker.SetConfig(cfg) }

// SetRouteBreakerConfig retunes the per-route breaker at runtime.
func (s *Service) SetRouteBreakerConfig(threshold int, cooldown time.Duration) {
	s.routeHealth.setThresholds(threshold, cooldown)
}

// BreakerConfig returns the endpoint breaker's current tuning.
func (s *Service) BreakerConfig() breaker.Config { return s.breaker.Config() }

// RouteBreakerConfig returns the route breaker's current tuning.
func (s *Service) RouteBreakerConfig() (int, time.Duration) { return s.routeHealth.thresholds() }

// killed reports whether the emergency kill switch disables a use case.
func (s *Service) killed(uc domain.UseCase) bool {
	return s.cfg.KillSwitch != nil && s.cfg.KillSwitch(uc)
}

// InvalidateRoutes drops the cached routes (call after admin changes).
func (s *Service) InvalidateRoutes() { s.repo.InvalidateCache() }

// clientFor builds a client for a route's endpoint, decrypting its key.
func (s *Service) clientFor(rt *domain.AIRoute) (*client.Client, string, error) {
	ep := rt.Endpoint
	if ep == nil {
		return nil, "", fmt.Errorf("route %s has no endpoint", rt.ID)
	}
	key, err := decryptEndpointKey(ep.APIKeyEncrypted, s.cfg.EncryptionKey)
	if err != nil {
		return nil, "", err
	}
	c := client.New(client.Config{
		BaseURL: ep.BaseURL,
		APIKey:  key,
		Timeout: s.cfg.DefaultTimeout,
	})
	return c, rt.Model, nil
}

func routeTimeout(rt *domain.AIRoute, def time.Duration) time.Duration {
	if rt.TimeoutMS > 0 {
		return time.Duration(rt.TimeoutMS) * time.Millisecond
	}
	return def
}

// eligibleRoutes returns routes whose endpoint is administratively active and
// whose breakers (endpoint AND route) allow traffic.
//
// Two levels of protection:
//   - endpoint breaker: the whole upstream is down (connection refused, 5xx for
//     everything) -> skip every route on it.
//   - route breaker: only this (endpoint, model) pair is dead -> skip just this
//     route; healthy siblings on the same endpoint keep serving.
func (s *Service) eligibleRoutes(routes []*domain.AIRoute) []*domain.AIRoute {
	var out []*domain.AIRoute
	for _, rt := range routes {
		if rt.Endpoint == nil || rt.Endpoint.Status != "active" {
			continue
		}
		if !s.breaker.Allow(rt.Endpoint.ID.String()) {
			continue
		}
		if !s.routeAllowed(rt) {
			continue
		}
		out = append(out, rt)
	}
	return out
}

// orderedRoutes filters eligible routes and orders them:
// 1. Groups eligible routes by normalized combo name ("default" when empty).
// 2. Selects combo ordering via Nginx Smooth Weighted Round-Robin (SWRR) based on
//    the aggregated endpoint weights in each combo.
// 3. Within each combo, routes are ordered by Priority ASC, then Endpoint Priority ASC.
// 4. The winning combo is placed first; if its routes fail, the remaining combos
//    are tried in descending order of remaining SWRR weight (failover).
func (s *Service) orderedRoutes(useCase domain.UseCase, routes []*domain.AIRoute) []*domain.AIRoute {
	eligible := s.eligibleRoutes(routes)
	if len(eligible) <= 1 {
		return eligible
	}

	comboRoutes := make(map[string][]*domain.AIRoute)
	comboWeights := make(map[string]int64)

	for _, rt := range eligible {
		combo := strings.TrimSpace(rt.Combo)
		if combo == "" {
			combo = "default"
		}
		comboRoutes[combo] = append(comboRoutes[combo], rt)

		w := int64(100)
		if rt.Endpoint != nil && rt.Endpoint.Weight > 0 {
			w = int64(rt.Endpoint.Weight)
		}
		comboWeights[combo] += w
	}

	// Sort routes within each combo by Priority ASC, then Endpoint Priority ASC.
	for _, cr := range comboRoutes {
		sort.SliceStable(cr, func(i, j int) bool {
			if cr[i].Priority != cr[j].Priority {
				return cr[i].Priority < cr[j].Priority
			}
			if cr[i].Endpoint != nil && cr[j].Endpoint != nil {
				return cr[i].Endpoint.Priority < cr[j].Endpoint.Priority
			}
			return false
		})
	}

	if len(comboRoutes) == 1 {
		for _, cr := range comboRoutes {
			return cr
		}
	}

	s.rrMu.Lock()
	state := s.rrState[useCase]
	if state == nil {
		state = make(map[string]int64)
		s.rrState[useCase] = state
	}

	var totalWeight int64
	combos := make([]string, 0, len(comboWeights))
	for c, w := range comboWeights {
		totalWeight += w
		combos = append(combos, c)
		if _, exists := state[c]; !exists {
			state[c] = rand.Int64N(w + 1)
		}
		state[c] += w
	}
	sort.Strings(combos)

	var bestCombo string
	var maxWeight int64 = -1 << 62
	for _, c := range combos {
		if state[c] > maxWeight {
			maxWeight = state[c]
			bestCombo = c
		}
	}
	state[bestCombo] -= totalWeight

	remaining := make([]string, 0, len(combos)-1)
	for _, c := range combos {
		if c != bestCombo {
			remaining = append(remaining, c)
		}
	}
	sort.SliceStable(remaining, func(i, j int) bool {
		if state[remaining[i]] != state[remaining[j]] {
			return state[remaining[i]] > state[remaining[j]]
		}
		return remaining[i] < remaining[j]
	})
	s.rrMu.Unlock()

	orderedCombos := append([]string{bestCombo}, remaining...)

	out := make([]*domain.AIRoute, 0, len(eligible))
	for _, c := range orderedCombos {
		out = append(out, comboRoutes[c]...)
	}
	return out
}

// ChatStream streams a chat completion through the failover chain.
func (s *Service) ChatStream(ctx context.Context, req *domain.AIChatRequest) (<-chan domain.AIChatStreamChunk, error) {
	useCase := req.UseCase
	if useCase == "" {
		useCase = domain.UseCaseChat
	}
	if s.killed(useCase) {
		return nil, ErrKillSwitch
	}
	routes, err := s.repo.GetActiveRoutesByUseCase(ctx, useCase)
	if err != nil {
		return nil, err
	}
	if len(routes) == 0 {
		return nil, domain.ErrNoRoute
	}

	var lastErr error
	for _, rt := range s.orderedRoutes(useCase, routes) {
		c, model, cerr := s.clientFor(rt)
		if cerr != nil {
			lastErr = cerr
			continue
		}
		if model == "" {
			model = rt.Endpoint.Provider
		}
		ch, serr := c.ChatStream(ctx, model, req, routeTimeout(rt, s.cfg.DefaultTimeout))
		if serr != nil {
			s.recordFailure(rt, serr)
			lastErr = serr
			continue
		}
		s.recordSuccess(rt)
		return ch, nil
	}
	if lastErr == nil {
		lastErr = domain.ErrAllEndpointsExhausted
	}
	return nil, lastErr
}

// ChatSync performs a non-streaming chat through the failover chain.
func (s *Service) ChatSync(ctx context.Context, req *domain.AIChatRequest) (string, domain.TokenUsage, error) {
	useCase := req.UseCase
	if useCase == "" {
		useCase = domain.UseCaseChat
	}
	if s.killed(useCase) {
		return "", domain.TokenUsage{}, ErrKillSwitch
	}
	routes, err := s.repo.GetActiveRoutesByUseCase(ctx, useCase)
	if err != nil {
		return "", domain.TokenUsage{}, err
	}
	if len(routes) == 0 {
		return "", domain.TokenUsage{}, domain.ErrNoRoute
	}

	var lastErr error
	for _, rt := range s.orderedRoutes(useCase, routes) {
		c, model, cerr := s.clientFor(rt)
		if cerr != nil {
			lastErr = cerr
			continue
		}
		if model == "" {
			model = rt.Endpoint.Provider
		}
		content, usage, serr := c.ChatSync(ctx, model, req, routeTimeout(rt, s.cfg.DefaultTimeout))
		if serr != nil {
			s.recordFailure(rt, serr)
			lastErr = serr
			continue
		}
		s.recordSuccess(rt)
		usage = s.finalizeUsage(usage, useCase, buildPromptText(req), content)
		return content, usage, nil
	}
	if lastErr == nil {
		lastErr = domain.ErrAllEndpointsExhausted
	}
	return "", domain.TokenUsage{}, lastErr
}

// CreateEmbedding returns a 768-dim vector for text.
func (s *Service) CreateEmbedding(ctx context.Context, req *domain.AIEmbeddingRequest) ([]float32, domain.TokenUsage, error) {
	if s.killed(domain.UseCaseEmbedding) {
		return nil, domain.TokenUsage{}, ErrKillSwitch
	}
	routes, err := s.repo.GetActiveRoutesByUseCase(ctx, domain.UseCaseEmbedding)
	if err != nil {
		return nil, domain.TokenUsage{}, err
	}
	if len(routes) == 0 {
		return nil, domain.TokenUsage{}, domain.ErrNoRoute
	}

	var lastErr error
	for _, rt := range s.orderedRoutes(domain.UseCaseEmbedding, routes) {
		c, model, cerr := s.clientFor(rt)
		if cerr != nil {
			lastErr = cerr
			continue
		}
		if req.Model != "" {
			model = req.Model
		}
		vec, usage, eerr := c.Embedding(ctx, model, req.Text, 768, routeTimeout(rt, s.cfg.DefaultTimeout))
		if eerr != nil {
			s.recordFailure(rt, eerr)
			lastErr = eerr
			continue
		}
		// Strict dimension lock: the pgvector column is vector(768).
		if len(vec) != 768 {
			s.recordFailure(rt, domain.ErrInvalidEmbeddingDimension)
			lastErr = fmt.Errorf("%w: got %d", domain.ErrInvalidEmbeddingDimension, len(vec))
			continue
		}
		s.recordSuccess(rt)
		usage = s.finalizeUsage(usage, domain.UseCaseEmbedding, req.Text, "")
		return vec, usage, nil
	}
	if lastErr == nil {
		lastErr = domain.ErrAllEndpointsExhausted
	}
	return nil, domain.TokenUsage{}, lastErr
}

// TranscribeAudio transcribes audio to text.
func (s *Service) TranscribeAudio(ctx context.Context, req *domain.AIAudioTranscriptionRequest) (string, domain.TokenUsage, error) {
	if s.killed(domain.UseCaseVoice) {
		return "", domain.TokenUsage{}, ErrKillSwitch
	}
	routes, err := s.repo.GetActiveRoutesByUseCase(ctx, domain.UseCaseVoice)
	if err != nil {
		return "", domain.TokenUsage{}, err
	}
	if len(routes) == 0 {
		return "", domain.TokenUsage{}, domain.ErrNoRoute
	}

	var lastErr error
	for _, rt := range s.orderedRoutes(domain.UseCaseVoice, routes) {
		c, model, cerr := s.clientFor(rt)
		if cerr != nil {
			lastErr = cerr
			continue
		}
		if req.Model != "" {
			model = req.Model
		}
		text, terr := c.Transcribe(ctx, model, req.Filename, req.Audio, routeTimeout(rt, s.cfg.DefaultTimeout))
		if terr != nil {
			s.recordFailure(rt, terr)
			lastErr = terr
			continue
		}
		s.recordSuccess(rt)
		usage := metering.EstimateAudio(estimateAudioSeconds(len(req.Audio)), text)
		return text, usage, nil
	}
	if lastErr == nil {
		lastErr = domain.ErrAllEndpointsExhausted
	}
	return "", domain.TokenUsage{}, lastErr
}

// AnalyzeVision returns the model's textual analysis of an image.
func (s *Service) AnalyzeVision(ctx context.Context, req *domain.AIVisionRequest) (string, domain.TokenUsage, error) {
	chatReq := &domain.AIChatRequest{
		UseCase:     domain.UseCaseVision,
		Model:       req.Model,
		Messages:    []domain.AIChatMessage{{Role: "user", Content: req.Prompt}},
		ImageBase64: req.ImageBase64,
		ImageMIME:   req.ImageMIME,
	}
	content, usage, err := s.ChatSync(ctx, chatReq)
	if err != nil {
		return "", domain.TokenUsage{}, err
	}
	if usage.Source == metering.SourceHeuristic {
		usage = metering.EstimateVision(req.Prompt, content)
	}
	return content, usage, nil
}

// ─── internals ────────────────────────────────────────────────

// recordSuccess clears both breaker windows for a route that just succeeded:
// the endpoint breaker (endpoint is demonstrably up) and the route breaker
// (this model demonstrably works).
func (s *Service) recordSuccess(rt *domain.AIRoute) {
	if rt.Endpoint == nil {
		return
	}
	s.breaker.RecordSuccess(rt.Endpoint.ID.String())
	s.recordRouteSuccess(rt)
}

func (s *Service) recordFailure(rt *domain.AIRoute, err error) {
	if rt.Endpoint == nil {
		return
	}
	// Always quarantine the specific route: whatever went wrong, this
	// (endpoint, model) pair just failed, so fail over past it.
	s.recordRouteFailure(rt, err.Error())

	// Only trip the ENDPOINT breaker when the whole endpoint looks down. A
	// model-level failure (4xx/5xx for one model, e.g. 503 "no such model")
	// must NOT quarantine the endpoint, or it would take down healthy sibling
	// models sharing that endpoint — the exact bug this file documents.
	if isEndpointLevelFailure(err) {
		s.breaker.RecordFailure(rt.Endpoint.ID.String())
	}

	logger.Warn("AI gateway route failed, attempting failover",
		zap.String("endpoint", rt.Endpoint.Name),
		zap.String("use_case", string(rt.UseCase)),
		zap.String("model", rt.Model),
		zap.Bool("endpoint_level", isEndpointLevelFailure(err)),
		zap.Error(err))
}

// isEndpointLevelFailure reports whether an error indicates the whole endpoint
// is unhealthy, as opposed to one model being unavailable.
//
// Evidence-based rule (measured against the local 9router):
//   - An HTTP response of ANY status means the endpoint is up and answering, so
//     it is NOT endpoint-level. This matters because 9router returns HTTP 503
//     for a dead model ("gemini-2.5-flash" -> 503 with a 404 body), so treating
//     5xx as "endpoint down" would quarantine healthy sibling models sharing the
//     endpoint — the exact bug this feature fixes.
//   - A timeout means one model was slow (e.g. a reasoning model), not that the
//     endpoint is down, so it is NOT endpoint-level either.
//   - Only a transport-level failure (connection refused, DNS, dial error) makes
//     the endpoint itself suspect.
func isEndpointLevelFailure(err error) bool {
	if err == nil {
		return false
	}
	// An HTTP response proves the endpoint is reachable.
	var ue *client.UpstreamError
	if errors.As(err, &ue) {
		return false
	}
	// A timeout is a slow model, not a dead endpoint.
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return false
	}
	if strings.Contains(strings.ToLower(err.Error()), "timeout") ||
		strings.Contains(strings.ToLower(err.Error()), "deadline exceeded") {
		return false
	}
	// Anything else (dial/refused/DNS/decrypt) makes the endpoint suspect.
	return true
}

// finalizeUsage fills in heuristic values when the provider omitted usage.
func (s *Service) finalizeUsage(usage domain.TokenUsage, useCase domain.UseCase, promptText, responseText string) domain.TokenUsage {
	if usage.TotalTokens > 0 {
		usage.Modality = string(useCase)
		return usage
	}
	return metering.Estimate(promptText, responseText, useCase)
}

func buildPromptText(req *domain.AIChatRequest) string {
	var b []byte
	for _, m := range req.Messages {
		b = append(b, m.Content...)
		b = append(b, '\n')
	}
	return string(b)
}

func estimateAudioSeconds(bytesLen int) float64 {
	// Rough heuristic for compressed audio: ~16KB per second.
	if bytesLen <= 0 {
		return 0
	}
	return float64(bytesLen) / 16000.0
}

func decryptEndpointKey(enc *string, masterKey []byte) (string, error) {
	if enc == nil || *enc == "" {
		return "", nil
	}
	if len(masterKey) != 32 {
		return "", errors.New("invalid encryption key length")
	}
	plain, err := crypto.Decrypt(*enc, masterKey)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// compile-time assertion.
var _ domain.AIGatewayService = (*Service)(nil)

// uuidRef keeps uuid import used by future admin helpers without pulling it in
// unnecessarily; referenced in UpdateEndpointHealth callers.
var _ = uuid.Nil
