package gateway

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/fintrack/backend/internal/aigateway/client"
	"github.com/fintrack/backend/internal/domain"
)

// RouteVerification is the outcome of probing one configured route's model.
type RouteVerification struct {
	UseCase  string `json:"use_case"`
	Model    string `json:"model"`
	Combo    string `json:"combo,omitempty"`
	Endpoint string `json:"endpoint"`
	// EndpointID is the endpoint UUID; used to key the route breaker so a broken
	// route can be quarantined by identity.
	EndpointID string `json:"endpoint_id,omitempty"`
	// OK is true when the configured model answered within the probe budget.
	OK bool `json:"ok"`
	// Skipped is true for use cases that cannot be probed without a fixture
	// (e.g. voice needs real audio); such routes are reported, never failed.
	Skipped bool `json:"skipped,omitempty"`
	// LatencyMS is how long the probe took.
	LatencyMS int64 `json:"latency_ms"`
	// Error is the upstream failure when OK is false (sanitized to a message).
	Error string `json:"error,omitempty"`
}

// verifyProbeCap bounds a single route probe. A dead model (e.g. HTTP 503)
// answers almost instantly, so this only limits how long we wait for a
// slow-but-alive model before reporting "timeout" instead of hanging.
const verifyProbeCap = 45 * time.Second

// probePNG is a 64x64 solid-blue PNG used to exercise vision routes without a
// real image fixture. It must be large enough for providers to decode — a 1x1
// image is rejected by some (measured: Google returned 400 "Unable to process
// input image"), which would report a working vision route as broken.
const probePNG = "iVBORw0KGgoAAAANSUhEUgAAAEAAAABACAIAAAAlC+aJAAAAT0lEQVR42u3PQQkAAAgEsEtiTsMaxgi+hcEKLNXzWgQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQELgsB1yFad2hFzQAAAABJRU5ErkJggg=="

// VerifyRoutes probes the ACTUAL model configured on every active route and
// reports which ones are broken. It is read-only: it never mutates routes,
// health, or the circuit breaker, so a false negative can never take AI down —
// it only surfaces the problem before a user hits it.
//
// This complements the endpoint reachability probe: an endpoint can be
// reachable ("healthy") while the specific model its route points at is dead
// (e.g. the gateway answers 503 for an unknown model). Reachability does not
// catch that; a real call does.
func (s *Service) VerifyRoutes(ctx context.Context) []RouteVerification {
	useCases := []domain.UseCase{
		domain.UseCaseChat,
		domain.UseCaseReasoning,
		domain.UseCaseVision,
		domain.UseCaseVoice,
		domain.UseCaseEmbedding,
	}

	var out []RouteVerification
	for _, uc := range useCases {
		routes, err := s.repo.GetActiveRoutesByUseCase(ctx, uc)
		if err != nil {
			out = append(out, RouteVerification{
				UseCase: string(uc),
				Error:   "failed to load routes: " + err.Error(),
			})
			continue
		}
		for _, rt := range routes {
			out = append(out, s.verifyRoute(ctx, rt))
		}
	}
	return out
}

// verifyRoute probes a single route's model.
func (s *Service) verifyRoute(ctx context.Context, rt *domain.AIRoute) RouteVerification {
	res := RouteVerification{
		UseCase: string(rt.UseCase),
		Model:   rt.Model,
		Combo:   rt.Combo,
	}
	if rt.Endpoint == nil {
		res.Error = "route has no endpoint"
		return res
	}
	res.Endpoint = rt.Endpoint.Name
	res.EndpointID = rt.Endpoint.ID.String()

	// Voice needs real audio to transcribe; there is no synthetic fixture that
	// is representative, so report it as skipped rather than failing it.
	if rt.UseCase == domain.UseCaseVoice {
		res.Skipped = true
		return res
	}

	c, model, err := s.clientFor(rt)
	if err != nil {
		res.Error = err.Error()
		return res
	}

	timeout := s.timeoutFor(rt)
	if timeout <= 0 || timeout > verifyProbeCap {
		timeout = verifyProbeCap
	}
	pctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	err = s.probeModel(pctx, c, rt.UseCase, model)
	res.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		res.Error = sanitizeVerifyError(err, pctx)
		return res
	}
	res.OK = true
	return res
}

// probeModel issues the smallest real call that proves a model is alive for the
// given use case.
func (s *Service) probeModel(ctx context.Context, c *client.Client, useCase domain.UseCase, model string) error {
	switch useCase {
	case domain.UseCaseEmbedding:
		vec, _, err := c.Embedding(ctx, model, "ping", 768, 0)
		if err != nil {
			return err
		}
		if len(vec) != 768 {
			return fmt.Errorf("embedding returned %d dims, want 768", len(vec))
		}
		return nil

	case domain.UseCaseVision:
		req := &domain.AIChatRequest{
			UseCase:     domain.UseCaseVision,
			Messages:    []domain.AIChatMessage{{Role: "user", Content: "What is in this image? One word."}},
			ImageBase64: probePNG,
			ImageMIME:   "image/png",
			MaxTokens:   16,
		}
		_, _, err := c.ChatSync(ctx, model, req, 0)
		return err

	default: // chat, reasoning
		req := &domain.AIChatRequest{
			UseCase:   useCase,
			Messages:  []domain.AIChatMessage{{Role: "user", Content: "Reply with the single word: OK"}},
			MaxTokens: 16,
		}
		_, _, err := c.ChatSync(ctx, model, req, 0)
		return err
	}
}

// sanitizeVerifyError turns a probe failure into a short, non-leaky message,
// distinguishing a timeout (the model may simply be slow) from a hard error
// (the model is likely dead or not served by the gateway).
func sanitizeVerifyError(err error, ctx context.Context) string {
	if ctx.Err() == context.DeadlineExceeded {
		return "timeout (model may be slow or unresponsive)"
	}
	msg := strings.TrimSpace(err.Error())
	// UpstreamError carries "upstream returned status NNN: <body>"; keep it short.
	if len(msg) > 200 {
		msg = msg[:200] + "…"
	}
	return msg
}

// decodeProbePNG is exposed for tests.
func decodeProbePNG() []byte {
	b, _ := base64.StdEncoding.DecodeString(probePNG)
	return b
}
