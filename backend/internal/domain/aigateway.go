package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ─── Use cases ────────────────────────────────────────────────

// UseCase identifies a class of AI work routed independently by the gateway.
type UseCase string

const (
	UseCaseChat      UseCase = "chat"
	UseCaseVision    UseCase = "vision"
	UseCaseVoice     UseCase = "voice"
	UseCaseEmbedding UseCase = "embedding"
	UseCaseReasoning UseCase = "reasoning"
)

// AllUseCases lists every supported use case.
var AllUseCases = []UseCase{UseCaseChat, UseCaseVision, UseCaseVoice, UseCaseEmbedding, UseCaseReasoning}

// ─── Entities ─────────────────────────────────────────────────

// AIEndpoint is a pooled upstream (9router proxy or a direct provider).
type AIEndpoint struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	Name            string     `json:"name" db:"name"`
	BaseURL         string     `json:"base_url" db:"base_url"`
	APIKeyEncrypted *string    `json:"-" db:"api_key_encrypted"`
	Provider        string     `json:"provider" db:"provider"`
	Priority        int        `json:"priority" db:"priority"`
	Status          string     `json:"status" db:"status"`
	Health          string     `json:"health" db:"health"`
	LastCheckAt     *time.Time `json:"last_check_at,omitempty" db:"last_check_at"`
	LastError       *string    `json:"last_error,omitempty" db:"last_error"`
	Modalities      []string   `json:"modalities" db:"modalities"`
	Weight          int        `json:"weight" db:"weight"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// AIRoute maps a use case to an endpoint + model, with a priority for failover.
type AIRoute struct {
	ID         uuid.UUID `json:"id" db:"id"`
	UseCase    UseCase   `json:"use_case" db:"use_case"`
	EndpointID uuid.UUID `json:"endpoint_id" db:"endpoint_id"`
	Model      string    `json:"model" db:"model"`
	Combo      string    `json:"combo" db:"combo"`
	Priority   int       `json:"priority" db:"priority"`
	IsActive   bool      `json:"is_active" db:"is_active"`
	TimeoutMS  int       `json:"timeout_ms" db:"timeout_ms"`
	MaxRetries int       `json:"max_retries" db:"max_retries"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`

	// Endpoint is populated by the repository join for convenience.
	Endpoint *AIEndpoint `json:"endpoint,omitempty"`
}

// ─── DTOs ─────────────────────────────────────────────────────

// AIChatMessage is a single conversation turn.
type AIChatMessage struct {
	Role    string `json:"role"` // system | user | assistant
	Content string `json:"content"`
}

// AIChatRequest is a chat/completion request routed through the gateway.
type AIChatRequest struct {
	UseCase     UseCase         `json:"use_case"`
	Model       string          `json:"model,omitempty"`
	Messages    []AIChatMessage `json:"messages"`
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	// ImageBase64, when set, is attached as a multimodal content part (vision).
	ImageBase64 string `json:"-"`
	ImageMIME   string `json:"-"`
}

// AIChatStreamChunk is one streamed delta.
type AIChatStreamChunk struct {
	Content string     `json:"content"`
	Done    bool       `json:"done"`
	Usage   TokenUsage `json:"usage,omitempty"`
}

// AIEmbeddingRequest asks for a vector embedding of a single text.
type AIEmbeddingRequest struct {
	Model string `json:"model,omitempty"`
	Text  string `json:"text"`
}

// AIEmbeddingResponse carries the vector and usage.
type AIEmbeddingResponse struct {
	Vector []float32  `json:"vector"`
	Usage  TokenUsage `json:"usage"`
}

// AIAudioTranscriptionRequest asks for audio transcription.
type AIAudioTranscriptionRequest struct {
	Model    string `json:"model,omitempty"`
	Filename string `json:"filename"`
	Audio    []byte `json:"-"`
	MIME     string `json:"-"`
}

// AIVisionRequest asks for structured analysis of an image (e.g. a receipt).
type AIVisionRequest struct {
	Model       string `json:"model,omitempty"`
	Prompt      string `json:"prompt"`
	ImageBase64 string `json:"-"`
	ImageMIME   string `json:"-"`
}

// TokenUsage reports token consumption for a single gateway call.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	// Source is "provider_reported" or "heuristic_estimated".
	Source string `json:"source"`
	// Modality records which use case produced the usage (for billing weights).
	Modality string `json:"modality,omitempty"`
}

// ─── Errors ───────────────────────────────────────────────────

var (
	// ErrAllEndpointsExhausted is returned when every route in the pool failed.
	ErrAllEndpointsExhausted = errors.New("all ai gateway endpoints exhausted")
	// ErrNoRoute indicates no active route exists for the requested use case.
	ErrNoRoute = errors.New("no active ai route for use case")
	// ErrInvalidEmbeddingDimension is returned when a vector is not 768-dim.
	ErrInvalidEmbeddingDimension = errors.New("invalid embedding dimension: expected 768")
	// ErrEndpointInactive indicates the endpoint is disabled (kill switch).
	ErrEndpointInactive = errors.New("ai endpoint is inactive")
	// ErrUpstream is a generic upstream failure (5xx/timeout/connection).
	ErrUpstream = errors.New("ai upstream failure")
)

// ─── Interfaces ───────────────────────────────────────────────

// AIEndpointRepository persists and reads pooled endpoints.
type AIEndpointRepository interface {
	GetActiveRoutesByUseCase(ctx context.Context, useCase UseCase) ([]*AIRoute, error)
	GetByID(ctx context.Context, id uuid.UUID) (*AIEndpoint, error)
	ListEndpoints(ctx context.Context) ([]*AIEndpoint, error)
	UpdateEndpointHealth(ctx context.Context, id uuid.UUID, health string, lastCheck time.Time, lastErr *string) error
	InvalidateCache()
}

// AIGatewayService is the transport-agnostic entry point used by business code.
type AIGatewayService interface {
	// ChatStream streams chat deltas; the returned channel closes when done.
	ChatStream(ctx context.Context, req *AIChatRequest) (<-chan AIChatStreamChunk, error)
	// ChatSync performs a non-streaming chat completion.
	ChatSync(ctx context.Context, req *AIChatRequest) (string, TokenUsage, error)
	// CreateEmbedding returns a 768-dim vector for text.
	CreateEmbedding(ctx context.Context, req *AIEmbeddingRequest) ([]float32, TokenUsage, error)
	// TranscribeAudio transcribes audio to text.
	TranscribeAudio(ctx context.Context, req *AIAudioTranscriptionRequest) (string, TokenUsage, error)
	// AnalyzeVision returns the model's textual analysis of an image.
	AnalyzeVision(ctx context.Context, req *AIVisionRequest) (string, TokenUsage, error)
}

// CircuitBreaker tracks per-endpoint failure state and gates requests.
type CircuitBreaker interface {
	// Allow reports whether a request may proceed for the endpoint.
	Allow(endpointID string) bool
	// RecordSuccess records a successful call.
	RecordSuccess(endpointID string)
	// RecordFailure records a failed call.
	RecordFailure(endpointID string)
	// State returns the current state name ("closed" | "open" | "half_open").
	State(endpointID string) string
}

// HealthChecker runs periodic probes against endpoints.
type HealthChecker interface {
	Start(ctx context.Context)
	Stop()
}
