package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"strings"

	"github.com/fintrack/backend/internal/domain"
)

// sseChunk is the subset of an OpenAI streaming chunk we consume.
type sseChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// streamSSE reads an SSE body line by line, forwarding content deltas to out and
// emitting a final chunk carrying usage (when provided) with Done=true.
//
// It tolerates: "data: " prefixes, blank lines, "[DONE]" sentinels, and
// keep-alive comments (lines starting with ':').
func streamSSE(r io.Reader, out chan<- domain.AIChatStreamChunk) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var lastUsage *domain.TokenUsage

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}

		var chunk sseChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// Skip malformed chunks rather than aborting the whole stream.
			continue
		}

		if chunk.Usage != nil {
			lastUsage = &domain.TokenUsage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
				Source:           "provider_reported",
			}
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			out <- domain.AIChatStreamChunk{Content: chunk.Choices[0].Delta.Content}
		}
	}

	final := domain.AIChatStreamChunk{Done: true}
	if lastUsage != nil {
		final.Usage = *lastUsage
	}
	out <- final
}

// multipartBody builds an OpenAI-compatible audio transcription form body and
// returns the buffer plus its Content-Type (including the boundary).
func multipartBody(model, filename string, audio []byte) (*bytes.Buffer, string) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("model", model)
	if filename == "" {
		filename = "audio.webm"
	}
	fw, err := w.CreateFormFile("file", filename)
	if err == nil {
		_, _ = fw.Write(audio)
	}
	_ = w.Close()
	return &buf, w.FormDataContentType()
}
