package usecase

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/internal/aigateway/metering"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/pkg/logger"
)

// gatewayUseCaseForMode maps the user-selected assistant mode onto a gateway
// use case. "reasoning" traffic is routed to the dedicated reasoning route pool
// (e.g. deepseek-r1); everything else uses the standard chat pool.
func gatewayUseCaseForMode(mode domain.LLMMode) domain.UseCase {
	if mode == domain.LLMReasoning {
		return domain.UseCaseReasoning
	}
	return domain.UseCaseChat
}

// reasoningEffortForMode returns the `reasoning_effort` value forwarded upstream
// for the given mode. It is opt-in via AI_REASONING_EFFORT ("low"|"medium"|"high")
// because some OpenAI-compatible providers reject the field for models that do
// not support it. When unset, the mode still changes routing (use case + model)
// without sending any extra field — which keeps non-reasoning providers working.
func reasoningEffortForMode(mode domain.LLMMode) string {
	if mode != domain.LLMReasoning {
		return ""
	}
	return strings.TrimSpace(os.Getenv("AI_REASONING_EFFORT"))
}

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
	usage *domain.TokenUsage,
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

	// Mode → gateway use case. If reasoning is disabled by Super Admin, fallback to fast.
	mode := req.Mode
	if mode == domain.LLMReasoning && !u.isReasoningEnabled() {
		mode = domain.LLMFast
	}
	useCase := gatewayUseCaseForMode(mode)
	reasoningEffort := u.reasoningEffortValue()
	maxTokens := 0
	if useCase == domain.UseCaseReasoning {
		maxTokens = u.reasoningMaxTokensValue()
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
	onReasoning := func(r string) {
		if r != "" {
			msgChan <- domain.ChatStreamEvent{Type: "reasoning", Reasoning: r}
		}
	}


	// First gateway call: stream the assistant response.
	var servedModel string
	responseText, callErr := u.gatewayStream(ctx, useCase, reasoningEffort, maxTokens, systemPrompt, history, req.Message, onContent, onReasoning, &servedModel, usage)
	flushStreamBuf()
	if callErr != nil {
		msgChan <- domain.ChatStreamEvent{Type: "content", Content: gatewayFriendlyError(callErr)}
		// The gateway could not serve the turn: the "content" above is a
		// sanitized error message, not an AI answer. Flag it so billing releases
		// the reservation instead of charging the user for unserved work.
		msgChan <- domain.ChatStreamEvent{Type: "done", Done: true, NoCharge: true}
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
				resp := u.generateToolResponse(results)
				fullResponse.WriteString(resp)
				msgChan <- domain.ChatStreamEvent{Type: "content", Content: resp}
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
		responseText, err = u.gatewayStream(ctx, useCase, reasoningEffort, maxTokens, systemPrompt, agentHistory, toolMessage, onContent, onReasoning, &servedModel, usage)
		flushStreamBuf()
		if err != nil {
			logger.Error("gateway agent loop failed", zap.Error(err))
			fullResponse.WriteString(fmt.Sprintf("\n\nMaaf, terjadi kesalahan saat memproses tool: %s", err.Error()))
			break
		}
	}

	if fullResponse.Len() == 0 {
		clean := strings.TrimSpace(stripToolCalls(responseText))
		if clean == "" {
			clean = strings.TrimSpace(responseText)
		}
		if clean == "" {
			clean = "Permintaan Anda telah berhasil diproses."
		}
		fullResponse.WriteString(clean)
	}

	// Persist messages + title using a detached context so cancellation of the
	// request (e.g. user clicked stop or connection closed) does not abort DB write.
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	if err := u.chatMessageRepo.Create(persistCtx, &domain.ChatMessage{SessionID: *sessionID, UserID: userID, Role: "user", Content: req.Message}); err != nil {
		logger.Error("failed to persist user chat message", zap.Error(err))
	}
	modeStr := string(req.Mode)
	if modeStr == "" {
		modeStr = string(domain.LLMFast)
	}
	assistantMsg := &domain.ChatMessage{SessionID: *sessionID, UserID: userID, Role: "assistant", Content: fullResponse.String(), Mode: &modeStr}
	if servedModel != "" {
		assistantMsg.ModelUsed = &servedModel
	}
	if err := u.chatMessageRepo.Create(persistCtx, assistantMsg); err != nil {
		logger.Error("failed to persist assistant chat message", zap.Error(err))
	}
	if session, _ := u.chatSessionRepo.GetByID(persistCtx, *sessionID, userID); session != nil && session.Title == "Percakapan baru" {
		if title := u.generateTitle(persistCtx, req.Message); title != "" && title != "Percakapan baru" {
			if err := u.chatSessionRepo.Update(persistCtx, &domain.ChatSession{ID: *sessionID, UserID: userID, Title: title}); err == nil {
				msgChan <- domain.ChatStreamEvent{Type: "title_update", Content: title}
			}
		}
	}

	var usagePtr *domain.TokenUsage
	if usage != nil && usage.TotalTokens > 0 {
		u := *usage
		usagePtr = &u
	}
	msgChan <- domain.ChatStreamEvent{Type: "done", Done: true, Mode: string(req.Mode), Model: servedModel, Usage: usagePtr}
	return nil
}

// gatewayStream issues one streaming chat call through the gateway and returns
// the full assistant text. It routes to the given use case (chat vs reasoning),
// forwards the optional reasoning effort, and records the concrete upstream
// model that served the request into servedModel (when non-nil).
func (u *AIUsecase) gatewayStream(
	ctx context.Context,
	useCase domain.UseCase,
	reasoningEffort string,
	maxTokens int,
	systemPrompt string,
	history []domain.ChatMessage,
	userMessage string,
	onContent func(string),
	onReasoning func(string),
	servedModel *string,
	usage *domain.TokenUsage,
) (string, error) {
	msgs := make([]domain.AIChatMessage, 0, len(history)+2)
	msgs = append(msgs, domain.AIChatMessage{Role: "system", Content: systemPrompt})
	for _, h := range history {
		msgs = append(msgs, domain.AIChatMessage{Role: h.Role, Content: h.Content})
	}
	msgs = append(msgs, domain.AIChatMessage{Role: "user", Content: userMessage})

	ch, err := u.gateway.ChatStream(ctx, &domain.AIChatRequest{
		UseCase:         useCase,
		Messages:        msgs,
		MaxTokens:       maxTokens,
		ReasoningEffort: reasoningEffort,
	})
	if err != nil {
		// Graceful degradation: if the reasoning pool cannot serve the request
		// (no active route / all endpoints down), fall back to the standard chat
		// pool rather than failing the turn. This is safe because ChatStream only
		// returns an error before any content has been streamed. We also drop the
		// reasoning_effort field, since chat models generally reject it.
		if useCase == domain.UseCaseReasoning {
			logger.Warn("reasoning route unavailable; falling back to chat pool", zap.Error(err))
			ch, err = u.gateway.ChatStream(ctx, &domain.AIChatRequest{
				UseCase:  domain.UseCaseChat,
				Messages: msgs,
			})
		}
		if err != nil {
			return "", err
		}
	}

	var sb strings.Builder
	// Streamed token usage reported by the provider on the terminal chunk
	// (IncludeUsage is requested upstream). When the provider omits it, the
	// metering fallback below derives a deterministic estimate.
	var streamUsage domain.TokenUsage
	for chunk := range ch {
		if chunk.Done {
			if chunk.Model != "" && servedModel != nil {
				*servedModel = chunk.Model
			}
			streamUsage = chunk.Usage
			break
		}
		if chunk.Content != "" {
			sb.WriteString(chunk.Content)
			onContent(chunk.Content)
		}
		if chunk.Reasoning != "" && onReasoning != nil {
			onReasoning(chunk.Reasoning)
		}
	}

	// Accumulate this call's usage into the caller's total. Provider-reported
	// values win; otherwise fall back to a heuristic estimate over the prompt and
	// response text so the ledger never charges a flat amount.
	if usage != nil {
		promptText := systemPrompt + "\n" + userMessage
		call := metering.FromProvider(
			streamUsage.PromptTokens, streamUsage.CompletionTokens, streamUsage.TotalTokens,
			useCase, promptText, sb.String(),
		)
		usage.PromptTokens += call.PromptTokens
		usage.CompletionTokens += call.CompletionTokens
		usage.TotalTokens += call.TotalTokens
		usage.Source = call.Source
		usage.Modality = call.Modality
	}
	return sb.String(), nil
}

// gatewayFriendlyError returns a user-facing message for gateway failures.
func gatewayFriendlyError(err error) string {
	return "Maaf, layanan AI sedang mengalami gangguan sementara. Silakan coba beberapa saat lagi."
}
