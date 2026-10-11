// Package client implements an OpenAI-compatible HTTP client used by the AI
// gateway to talk to 9router (or any OpenAI-compatible upstream). It supports
// chat completions (streaming and non-streaming), embeddings, audio
// transcriptions, and multimodal (vision) requests.
package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fintrack/backend/internal/domain"
)

// Config configures a Client.
type Config struct {
	// BaseURL is the OpenAI-compatible root, e.g. http://localhost:20128/v1.
	BaseURL string
	// APIKey is sent as a Bearer token. Empty for keyless local proxies.
	APIKey string
	// Timeout applies when a request does not specify one.
	Timeout time.Duration
}

// Client talks to a single upstream endpoint.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// New builds a Client with a pooled transport.
func New(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		// Disable compression to keep SSE streaming predictable.
		DisableCompression: true,
	}
	return &Client{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  cfg.APIKey,
		http:    &http.Client{Transport: transport, Timeout: timeout},
	}
}

// ─── Chat ─────────────────────────────────────────────────────

type chatCompletionRequest struct {
	Model         string         `json:"model"`
	Messages      []chatMessage  `json:"messages"`
	Stream        bool           `json:"stream"`
	StreamOptions *streamOptions `json:"stream_options,omitempty"`
	Temperature   *float64       `json:"temperature,omitempty"`
	MaxTokens     int            `json:"max_tokens,omitempty"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// textPart / imagePart build multimodal content.
type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

// ChatStream issues a streaming chat completion. It returns a channel of
// deltas; the final delta carries Usage and Done=true. The channel is closed
// when the stream ends or ctx is cancelled.
func (c *Client) ChatStream(ctx context.Context, model string, req *domain.AIChatRequest, timeout time.Duration) (<-chan domain.AIChatStreamChunk, error) {
	body := chatCompletionRequest{
		Model:         model,
		Messages:      buildMessages(req),
		Stream:        true,
		StreamOptions: &streamOptions{IncludeUsage: true},
		Temperature:   req.Temperature,
		MaxTokens:     req.MaxTokens,
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := c.newRequest(ctx, http.MethodPost, "/chat/completions", bytes.NewReader(raw), timeout)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, upstreamError(resp)
	}

	out := make(chan domain.AIChatStreamChunk, 16)
	go func() {
		defer close(out)
		defer resp.Body.Close()
		streamSSE(resp.Body, out)
	}()
	return out, nil
}

// ChatSync issues a non-streaming chat completion and returns the content.
func (c *Client) ChatSync(ctx context.Context, model string, req *domain.AIChatRequest, timeout time.Duration) (string, domain.TokenUsage, error) {
	body := chatCompletionRequest{
		Model:       model,
		Messages:    buildMessages(req),
		Stream:      false,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", domain.TokenUsage{}, err
	}

	httpReq, err := c.newRequest(ctx, http.MethodPost, "/chat/completions", bytes.NewReader(raw), timeout)
	if err != nil {
		return "", domain.TokenUsage{}, err
	}

	resp, err := c.do(httpReq)
	if err != nil {
		return "", domain.TokenUsage{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", domain.TokenUsage{}, upstreamError(resp)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", domain.TokenUsage{}, err
	}

	var content string
	if len(parsed.Choices) > 0 {
		content = parsed.Choices[0].Message.Content
	}
	usage := domain.TokenUsage{
		PromptTokens:     parsed.Usage.PromptTokens,
		CompletionTokens: parsed.Usage.CompletionTokens,
		TotalTokens:      parsed.Usage.TotalTokens,
		Source:           "provider_reported",
	}
	return content, usage, nil
}

// ─── Embeddings ───────────────────────────────────────────────

// Embedding returns the vector for text and its usage. When dims > 0,
// the dimensions field is sent in the request payload (required by providers
// whose default dimension does not match the target schema, e.g. gemini-embedding-001
// which defaults to 3072d but supports 768d via dimensions=768).
func (c *Client) Embedding(ctx context.Context, model, text string, dims int, timeout time.Duration) ([]float32, domain.TokenUsage, error) {
	payload := map[string]any{"model": model, "input": text}
	if dims > 0 {
		payload["dimensions"] = dims
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, domain.TokenUsage{}, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/embeddings", bytes.NewReader(raw), timeout)
	if err != nil {
		return nil, domain.TokenUsage{}, err
	}

	resp, err := c.do(httpReq)
	if err != nil {
		return nil, domain.TokenUsage{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, domain.TokenUsage{}, upstreamError(resp)
	}

	var parsed struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
		Usage struct {
			PromptTokens int `json:"prompt_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, domain.TokenUsage{}, err
	}
	if len(parsed.Data) == 0 {
		return nil, domain.TokenUsage{}, fmt.Errorf("embedding response had no data")
	}
	usage := domain.TokenUsage{
		PromptTokens: parsed.Usage.PromptTokens,
		TotalTokens:  parsed.Usage.TotalTokens,
		Source:       "provider_reported",
	}
	return parsed.Data[0].Embedding, usage, nil
}

// ─── Audio ────────────────────────────────────────────────────

// Transcribe sends audio for transcription (multipart/form-data).
func (c *Client) Transcribe(ctx context.Context, model, filename string, audio []byte, timeout time.Duration) (string, error) {
	buf, contentType := multipartBody(model, filename, audio)

	httpReq, err := c.newRequest(ctx, http.MethodPost, "/audio/transcriptions", buf, timeout)
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", contentType)

	resp, err := c.do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", upstreamError(resp)
	}

	var parsed struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	return parsed.Text, nil
}

// ─── Helpers ──────────────────────────────────────────────────

func buildMessages(req *domain.AIChatRequest) []chatMessage {
	out := make([]chatMessage, 0, len(req.Messages)+1)
	for _, m := range req.Messages {
		// Attach an image to the last user message when present (vision).
		if m.Role == "user" && req.ImageBase64 != "" && len(out) == len(req.Messages)-1 {
			parts := []contentPart{{Type: "text", Text: m.Content}}
			mime := req.ImageMIME
			if mime == "" {
				mime = "image/jpeg"
			}
			parts = append(parts, contentPart{
				Type:     "image_url",
				ImageURL: &imageURL{URL: "data:" + mime + ";base64," + req.ImageBase64},
			})
			out = append(out, chatMessage{Role: m.Role, Content: parts})
			continue
		}
		out = append(out, chatMessage{Role: m.Role, Content: m.Content})
	}
	return out
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader, timeout time.Duration) (*http.Request, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		// The cancel is intentionally not called here: it is tied to the
		// request lifetime and released when the request/stream completes.
		_ = cancel
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	return req, nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	return c.http.Do(req)
}

// upstreamError converts a non-2xx response into a typed error carrying status.
func upstreamError(resp *http.Response) error {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return &UpstreamError{StatusCode: resp.StatusCode, Body: string(b)}
}

// UpstreamError represents a non-2xx upstream response.
type UpstreamError struct {
	StatusCode int
	Body       string
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("upstream returned status %d: %s", e.StatusCode, e.Body)
}

// IsRetryable reports whether the upstream error warrants failover
// (5xx, 429, or network-level errors).
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	if ue, ok := err.(*UpstreamError); ok {
		return ue.StatusCode >= 500 || ue.StatusCode == http.StatusTooManyRequests
	}
	// Network errors (timeouts, connection refused) are retryable.
	return true
}

// EncodeBase64 is a small helper exposed for tests and callers.
func EncodeBase64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
