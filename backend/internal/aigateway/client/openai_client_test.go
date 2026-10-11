package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fintrack/backend/internal/domain"
)

// TestChatStream_ParsesSSE verifies streaming content + final usage.
func TestChatStream_ParsesSSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		chunks := []string{
			`data: {"choices":[{"delta":{"content":"Hello"}}]}`,
			`data: {"choices":[{"delta":{"content":" world"}}]}`,
			`data: {"choices":[{"delta":{}}],"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12}}`,
			`data: [DONE]`,
		}
		for _, c := range chunks {
			fmt.Fprintf(w, "%s\n\n", c)
			fl.Flush()
		}
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL + "/v1"})
	ch, err := c.ChatStream(context.Background(), "gpt-4o-mini", &domain.AIChatRequest{
		Messages: []domain.AIChatMessage{{Role: "user", Content: "hi"}},
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("stream: %v", err)
	}

	var content strings.Builder
	var final domain.AIChatStreamChunk
	for chunk := range ch {
		if chunk.Done {
			final = chunk
			break
		}
		content.WriteString(chunk.Content)
	}

	if content.String() != "Hello world" {
		t.Fatalf("expected 'Hello world', got %q", content.String())
	}
	if !final.Done {
		t.Fatal("expected final Done chunk")
	}
	if final.Usage.TotalTokens != 12 || final.Usage.Source != "provider_reported" {
		t.Fatalf("unexpected usage: %+v", final.Usage)
	}
}

// TestChatSync_ReturnsContentAndUsage verifies non-streaming chat.
func TestChatSync_ReturnsContentAndUsage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"content": "hai"}}},
			"usage":   map[string]any{"prompt_tokens": 5, "completion_tokens": 3, "total_tokens": 8},
		})
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL + "/v1"})
	content, usage, err := c.ChatSync(context.Background(), "gpt-4o-mini", &domain.AIChatRequest{
		Messages: []domain.AIChatMessage{{Role: "user", Content: "hi"}},
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if content != "hai" {
		t.Fatalf("expected hai, got %q", content)
	}
	if usage.TotalTokens != 8 {
		t.Fatalf("expected 8 tokens, got %d", usage.TotalTokens)
	}
}

// TestEmbedding_ReturnsVector verifies the embeddings endpoint parsing.
func TestEmbedding_ReturnsVector(t *testing.T) {
	vec := make([]float32, 768)
	for i := range vec {
		vec[i] = float32(i) / 768.0
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":  []any{map[string]any{"embedding": vec}},
			"usage": map[string]any{"prompt_tokens": 4, "total_tokens": 4},
		})
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL + "/v1"})
	got, usage, err := c.Embedding(context.Background(), "text-embedding-004", "halo", 768, 5*time.Second)
	if err != nil {
		t.Fatalf("embedding: %v", err)
	}
	if len(got) != 768 {
		t.Fatalf("expected 768 dims, got %d", len(got))
	}
	if usage.TotalTokens != 4 {
		t.Fatalf("expected 4 tokens, got %d", usage.TotalTokens)
	}
}

func TestEmbedding_SendsDimensionsPayload(t *testing.T) {
	var capturedPayload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPayload = make(map[string]any)
		_ = json.NewDecoder(r.Body).Decode(&capturedPayload)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":  []any{map[string]any{"embedding": []float32{0.1, 0.2}}},
			"usage": map[string]any{"total_tokens": 2},
		})
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL + "/v1"})

	// Case 1: dims > 0 -> dimensions included in JSON
	_, _, err := c.Embedding(context.Background(), "gemini-embedding-001", "halo", 768, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if d, ok := capturedPayload["dimensions"]; !ok || d != float64(768) {
		t.Errorf("expected dimensions=768 in payload, got %v", capturedPayload)
	}

	// Case 2: dims == 0 -> dimensions omitted from JSON
	_, _, err = c.Embedding(context.Background(), "text-embedding-004", "halo", 0, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, ok := capturedPayload["dimensions"]; ok {
		t.Errorf("dimensions field must be omitted when dims=0, got %v", capturedPayload)
	}
}

// TestTranscribe_ReturnsText verifies multipart transcription.
func TestTranscribe_ReturnsText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("expected multipart, got %s", r.Header.Get("Content-Type"))
		}
		_ = r.ParseMultipartForm(1 << 20)
		if r.FormValue("model") != "whisper-1" {
			t.Errorf("expected model whisper-1, got %s", r.FormValue("model"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"text": "transkripsi"})
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL + "/v1"})
	text, err := c.Transcribe(context.Background(), "whisper-1", "a.webm", []byte("audio"), 5*time.Second)
	if err != nil {
		t.Fatalf("transcribe: %v", err)
	}
	if text != "transkripsi" {
		t.Fatalf("expected transkripsi, got %q", text)
	}
}

// TestUpstreamError_Retryable verifies status classification.
func TestUpstreamError_Retryable(t *testing.T) {
	cases := []struct {
		status int
		want   bool
	}{
		{500, true}, {502, true}, {503, true}, {429, true},
		{400, false}, {404, false}, {401, false},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
		}))
		c := New(Config{BaseURL: srv.URL})
		_, _, err := c.ChatSync(context.Background(), "m", &domain.AIChatRequest{}, time.Second)
		if got := IsRetryable(err); got != tc.want {
			t.Errorf("status %d: IsRetryable=%v want %v (err=%v)", tc.status, got, tc.want, err)
		}
		srv.Close()
	}
}

// TestChatStream_AuthorizationHeader verifies the bearer token is sent.
func TestChatStream_AuthorizationHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret-key" {
			t.Errorf("expected bearer token, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL + "/v1", APIKey: "secret-key"})
	ch, err := c.ChatStream(context.Background(), "m", &domain.AIChatRequest{}, time.Second)
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	for range ch {
	}
}
