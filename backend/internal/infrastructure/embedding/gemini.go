package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GeminiEmbedder calls the Gemini REST embeddings endpoint
// (text-embedding-004, 768 dimensions). It deliberately avoids the SDK and
// mirrors the REST style already used elsewhere in the codebase.
type GeminiEmbedder struct {
	apiKey  string
	model   string
	client  *http.Client
	baseURL string
}

// NewGeminiEmbedder builds an embedder. If apiKey is empty the embedder is
// disabled: Embed returns ErrDisabled and callers should skip RAG.
func NewGeminiEmbedder(apiKey string) *GeminiEmbedder {
	return &GeminiEmbedder{
		apiKey:  apiKey,
		model:   "text-embedding-004",
		client:  &http.Client{Timeout: 10 * time.Second},
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
	}
}

// ErrDisabled indicates no embedding API key is configured.
var ErrDisabled = fmt.Errorf("embedding disabled: GEMINI_EMBEDDING_API_KEY is not set")

// Enabled reports whether an API key is configured.
func (e *GeminiEmbedder) Enabled() bool { return e.apiKey != "" }

type embedRequest struct {
	Model   string      `json:"model"`
	Content embedContent `json:"content"`
}

type embedContent struct {
	Parts []embedPart `json:"parts"`
}

type embedPart struct {
	Text string `json:"text"`
}

type embedResponse struct {
	Embedding struct {
		Values []float32 `json:"values"`
	} `json:"embedding"`
}

// Embed returns the 768-dim embedding of text. Retries once on 5xx / 429.
func (e *GeminiEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if !e.Enabled() {
		return nil, ErrDisabled
	}

	body, err := json.Marshal(embedRequest{
		Model:   "models/" + e.model,
		Content: embedContent{Parts: []embedPart{{Text: text}}},
	})
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/models/%s:embedContent?key=%s", e.baseURL, e.model, e.apiKey)

	var lastErr error
	for attempt := range 2 {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := e.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("embedding request failed: status %d: %s", resp.StatusCode, string(respBody))
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("embedding request failed: status %d: %s", resp.StatusCode, string(respBody))
		}

		var parsed embedResponse
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return nil, fmt.Errorf("failed to parse embedding response: %w", err)
		}
		if len(parsed.Embedding.Values) == 0 {
			return nil, fmt.Errorf("embedding response had no values")
		}
		return parsed.Embedding.Values, nil
	}

	return nil, lastErr
}
