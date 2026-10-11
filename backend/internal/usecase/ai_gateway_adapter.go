package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/pkg/logger"
)

// chatViaGateway runs the Chat flow through the centralized AI gateway (P03).
// It reuses the same agent-loop semantics as the legacy path but obtains the
// model response from the pooled endpoints instead of per-user keys.
func (u *AIUsecase) chatViaGateway(
	ctx context.Context,
	userID uuid.UUID,
	req domain.ChatRequest,
	sessionID *uuid.UUID,
	systemPrompt string,
	msgChan chan<- domain.ChatStreamEvent,
) error {
	history, _ := u.chatMessageRepo.ListBySession(ctx, *sessionID, userID)
	if len(history) > 20 {
		history = history[len(history)-20:]
	}

	msgChan <- domain.ChatStreamEvent{
		Type:       "agent_step",
		StepLabel:  "Menganalisis perintah...",
		StepStatus: "success",
	}

	// Streaming filter: strip <TOOL_CALL>/<TOOL_RESULT> blocks so raw tool JSON
	// is never exposed to the client. Identical to the legacy path.
	var streamBuf string
	var insideToolTag bool
	var curTagName string
	onContent := func(chunk string) {
		streamBuf += chunk
		for {
			if insideToolTag {
				endTag := "</" + curTagName + ">"
				endIdx := strings.Index(streamBuf, endTag)
				if endIdx == -1 {
					break
				}
				streamBuf = streamBuf[endIdx+len(endTag):]
				insideToolTag = false
				continue
			}
			callIdx := strings.Index(streamBuf, "<TOOL_CALL>")
			resultIdx := strings.Index(streamBuf, "<TOOL_RESULT>")
			tagIdx := -1
			tagLen := 0
			tagName := ""
			if callIdx >= 0 && (resultIdx < 0 || callIdx < resultIdx) {
				tagIdx = callIdx
				tagLen = len("<TOOL_CALL>")
				tagName = "TOOL_CALL"
			} else if resultIdx >= 0 {
				tagIdx = resultIdx
				tagLen = len("<TOOL_RESULT>")
				tagName = "TOOL_RESULT"
			}
			if tagIdx == -1 {
				ltIdx := strings.LastIndex(streamBuf, "<TOOL_")
				if ltIdx >= 0 && ltIdx > len(streamBuf)-20 {
					if safe := streamBuf[:ltIdx]; safe != "" {
						msgChan <- domain.ChatStreamEvent{Type: "content", Content: safe}
					}
					streamBuf = streamBuf[ltIdx:]
				} else if streamBuf != "" {
					msgChan <- domain.ChatStreamEvent{Type: "content", Content: streamBuf}
					streamBuf = ""
				}
				break
			}
			if tagIdx > 0 {
				msgChan <- domain.ChatStreamEvent{Type: "content", Content: streamBuf[:tagIdx]}
			}
			streamBuf = streamBuf[tagIdx+tagLen:]
			insideToolTag = true
			curTagName = tagName
		}
	}
	flushStreamBuf := func() {
		if !insideToolTag && streamBuf != "" {
			msgChan <- domain.ChatStreamEvent{Type: "content", Content: streamBuf}
			streamBuf = ""
		}
	}

	// First gateway call: stream the assistant response.
	responseText, callErr := u.gatewayStream(ctx, systemPrompt, history, req.Message, onContent)
	flushStreamBuf()
	if callErr != nil {
		msgChan <- domain.ChatStreamEvent{Type: "content", Content: gatewayFriendlyError(callErr)}
		msgChan <- domain.ChatStreamEvent{Type: "done", Done: true}
		return nil
	}

	// Agent loop (max 2 iterations), identical semantics to the legacy path.
	var fullResponse strings.Builder
	agentHistory := make([]domain.ChatMessage, 0, len(history)+4)
	agentHistory = append(agentHistory, history...)
	agentHistory = append(agentHistory, domain.ChatMessage{
		SessionID: *sessionID, UserID: userID, Role: "user", Content: req.Message,
	})

	for range 2 {
		toolCalls := parseToolCalls(ctx, responseText)
		if len(toolCalls) == 0 {
			fullResponse.WriteString(responseText)
			break
		}
		cleanText := stripToolCalls(responseText)

		var results []toolExecResult
		for _, call := range toolCalls {
			msgChan <- domain.ChatStreamEvent{Type: "agent_step", StepLabel: u.getToolLabel(ctx, call.Tool), StepStatus: "running"}
			msgChan <- domain.ChatStreamEvent{Type: "tool_call", ToolCall: &domain.ToolCallInfo{Tool: call.Tool, Parameters: nil}}
			result := u.executeTool(ctx, userID, call)
			results = append(results, result)
			var sanitizedData interface{}
			if result.Tool == "generate_chart" && result.Data != nil {
				sanitizedData = result.Data
			}
			msgChan <- domain.ChatStreamEvent{Type: "tool_result", ToolResult: &domain.ToolResultInfo{Tool: result.Tool, Data: sanitizedData, Error: result.Error}}
			status := "success"
			if result.Error != "" {
				status = "error"
			}
			msgChan <- domain.ChatStreamEvent{Type: "agent_step", StepLabel: u.getToolLabel(ctx, call.Tool), StepStatus: status}
		}

		if allToolsSimple(results) {
			cleanText = strings.TrimSpace(cleanText)
			if cleanText != "" {
				fullResponse.WriteString(cleanText)
			} else {
				fullResponse.WriteString(u.generateToolResponse(results))
			}
			break
		}

		agentHistory = append(agentHistory, domain.ChatMessage{
			SessionID: *sessionID, UserID: userID, Role: "assistant", Content: responseText,
		})
		if len(agentHistory) > 12 {
			agentHistory = agentHistory[len(agentHistory)-12:]
		}

		toolMessage := u.buildToolResultPrompt(results)
		var err error
		responseText, err = u.gatewayStream(ctx, systemPrompt, agentHistory, toolMessage, onContent)
		flushStreamBuf()
		if err != nil {
			logger.Error("gateway agent loop failed", zap.Error(err))
			fullResponse.WriteString(fmt.Sprintf("\n\nMaaf, terjadi kesalahan saat memproses tool: %s", err.Error()))
			break
		}
	}

	// Persist messages + title, mirroring the legacy path.
	_ = u.chatMessageRepo.Create(ctx, &domain.ChatMessage{SessionID: *sessionID, UserID: userID, Role: "user", Content: req.Message})
	_ = u.chatMessageRepo.Create(ctx, &domain.ChatMessage{SessionID: *sessionID, UserID: userID, Role: "assistant", Content: fullResponse.String()})

	if session, _ := u.chatSessionRepo.GetByID(ctx, *sessionID, userID); session != nil && session.Title == "Percakapan baru" {
		if title := u.generateTitle(ctx, req.Message); title != "" && title != "Percakapan baru" {
			if err := u.chatSessionRepo.Update(ctx, &domain.ChatSession{ID: *sessionID, UserID: userID, Title: title}); err == nil {
				msgChan <- domain.ChatStreamEvent{Type: "title_update", Content: title}
			}
		}
	}

	msgChan <- domain.ChatStreamEvent{Type: "done", Done: true}
	return nil
}

// gatewayStream issues one streaming chat call through the gateway and returns
// the full assistant text.
func (u *AIUsecase) gatewayStream(ctx context.Context, systemPrompt string, history []domain.ChatMessage, userMessage string, onContent func(string)) (string, error) {
	msgs := make([]domain.AIChatMessage, 0, len(history)+2)
	msgs = append(msgs, domain.AIChatMessage{Role: "system", Content: systemPrompt})
	for _, h := range history {
		msgs = append(msgs, domain.AIChatMessage{Role: h.Role, Content: h.Content})
	}
	msgs = append(msgs, domain.AIChatMessage{Role: "user", Content: userMessage})

	ch, err := u.gateway.ChatStream(ctx, &domain.AIChatRequest{UseCase: domain.UseCaseChat, Messages: msgs})
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for chunk := range ch {
		if chunk.Done {
			break
		}
		if chunk.Content != "" {
			sb.WriteString(chunk.Content)
			onContent(chunk.Content)
		}
	}
	return sb.String(), nil
}

// gatewayFriendlyError returns a user-facing message for gateway failures.
func gatewayFriendlyError(err error) string {
	return "Maaf, layanan AI sedang mengalami gangguan sementara. Silakan coba beberapa saat lagi."
}
