// Package gateway wires the pieces of the centralized AI gateway together: a
// per-use-case route pool (repository), a circuit breaker, an OpenAI-compatible
// client, and token metering. It exposes the domain.AIGatewayService interface
// used by business code.
package gateway

import (
	"context"
	"errors"
	"fmt"
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
}

// ErrKillSwitch is returned when the emergency kill switch disables a use case.
var ErrKillSwitch = errors.New("ai gateway disabled by kill switch")

// Service implements domain.AIGatewayService over a pooled set of endpoints.
type Service struct {
	repo    domain.AIEndpointRepository
	breaker *breaker.Breaker
	cfg     Config
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
	return &Service{
		repo:    repo,
		breaker: breaker.New(bc),
		cfg:     cfg,
	}
}

// Breaker exposes the breaker (for health checks / diagnostics).
func (s *Service) Breaker() *breaker.Breaker { return s.breaker }

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
// whose breaker allows traffic.
func (s *Service) eligibleRoutes(routes []*domain.AIRoute) []*domain.AIRoute {
	var out []*domain.AIRoute
	for _, rt := range routes {
		if rt.Endpoint == nil || rt.Endpoint.Status != "active" {
			continue
		}
		if !s.breaker.Allow(rt.Endpoint.ID.String()) {
			continue
		}
		out = append(out, rt)
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
	for _, rt := range s.eligibleRoutes(routes) {
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
		s.breaker.RecordSuccess(rt.Endpoint.ID.String())
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
	for _, rt := range s.eligibleRoutes(routes) {
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
		s.breaker.RecordSuccess(rt.Endpoint.ID.String())
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
	for _, rt := range s.eligibleRoutes(routes) {
		c, model, cerr := s.clientFor(rt)
		if cerr != nil {
			lastErr = cerr
			continue
		}
		if req.Model != "" {
			model = req.Model
		}
		vec, usage, eerr := c.Embedding(ctx, model, req.Text, routeTimeout(rt, s.cfg.DefaultTimeout))
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
		s.breaker.RecordSuccess(rt.Endpoint.ID.String())
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
	for _, rt := range s.eligibleRoutes(routes) {
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
		s.breaker.RecordSuccess(rt.Endpoint.ID.String())
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

func (s *Service) recordFailure(rt *domain.AIRoute, err error) {
	if rt.Endpoint == nil {
		return
	}
	s.breaker.RecordFailure(rt.Endpoint.ID.String())
	logger.Warn("AI gateway route failed, attempting failover",
		zap.String("endpoint", rt.Endpoint.Name),
		zap.String("use_case", string(rt.UseCase)),
		zap.Error(err))
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
