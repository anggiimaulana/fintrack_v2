package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ChatSession struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Title     string    `json:"title" db:"title"`
	Source    string    `json:"source" db:"source"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type ChatMessage struct {
	ID         uuid.UUID `json:"id" db:"id"`
	SessionID  uuid.UUID `json:"session_id" db:"session_id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	Role       string    `json:"role" db:"role"`
	Content    string    `json:"content" db:"content"`
	ModelUsed  *string   `json:"model_used,omitempty" db:"model_used"`
	Mode       *string   `json:"mode,omitempty" db:"mode"`
	TokensUsed *int      `json:"tokens_used,omitempty" db:"tokens_used"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type AIProvider struct {
	ID                uuid.UUID `json:"id" db:"id"`
	Name              string    `json:"name" db:"name"`
	DisplayName       string    `json:"display_name" db:"display_name"`
	SupportsFast      bool      `json:"supports_fast" db:"supports_fast"`
	SupportsReasoning bool      `json:"supports_reasoning" db:"supports_reasoning"`
	IsActive          bool      `json:"is_active" db:"is_active"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

type CreateProviderRequest struct {
	Name              string `json:"name" binding:"required"`
	DisplayName       string `json:"display_name" binding:"required"`
	SupportsFast      bool   `json:"supports_fast"`
	SupportsReasoning bool   `json:"supports_reasoning"`
	IsActive          bool   `json:"is_active"`
}

type UpdateProviderRequest struct {
	Name              *string `json:"name"`
	DisplayName       *string `json:"display_name"`
	SupportsFast      *bool   `json:"supports_fast"`
	SupportsReasoning *bool   `json:"supports_reasoning"`
	IsActive          *bool   `json:"is_active"`
}

type LLMMode string

const (
	LLMFast      LLMMode = "fast"
	LLMReasoning LLMMode = "reasoning"
)

type ChatRequest struct {
	Message string  `json:"message" binding:"required"`
	Mode    LLMMode `json:"mode"`
}

type ToolCallInfo struct {
	Tool       string                 `json:"tool"`
	Parameters map[string]interface{} `json:"parameters"`
}

type ToolResultInfo struct {
	Tool  string      `json:"tool"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

type ChatStreamEvent struct {
	Type       string          `json:"type"`
	Content    string          `json:"content,omitempty"`
	Step       string          `json:"step,omitempty"`
	StepLabel  string          `json:"step_label,omitempty"`
	StepStatus string          `json:"step_status,omitempty"`
	Done       bool            `json:"done,omitempty"`
	// Mode echoes the requested assistant mode ("fast" | "reasoning") so the
	// client can label the answer it is about to receive.
	Mode string `json:"mode,omitempty"`
	// Model is the concrete upstream model that produced the reply (e.g.
	// "deepseek-r1"). Set on the terminal "done" event.
	Model string `json:"model,omitempty"`
	// Reasoning carries a chain-of-thought delta when the model exposes one.
	Reasoning  string          `json:"reasoning,omitempty"`
	ToolCall   *ToolCallInfo   `json:"tool_call,omitempty"`
	ToolResult *ToolResultInfo `json:"tool_result,omitempty"`
	// Usage carries the upstream token usage on the terminal "done" event so the
	// billing ledger can charge the real amount instead of a flat estimate.
	Usage *TokenUsage `json:"usage,omitempty"`
	// NoCharge marks a terminal event for a turn that consumed NO billable AI
	// work (kill switch, content filter, cooldown, gateway unavailable/error, or
	// a canned template answer). The billing layer releases the reserved quota
	// in that case: the user must not pay for a turn the AI never served.
	NoCharge bool `json:"no_charge,omitempty"`
}

type ChatSessionRepository interface {
	Create(ctx context.Context, session *ChatSession) error
	// CreateWithID inserts a session with a caller-supplied id, used to restore a
	// session that a client referenced but that no longer exists (so follow-up
	// messages do not violate the chat_messages session foreign key).
	CreateWithID(ctx context.Context, session *ChatSession) error
	GetByID(ctx context.Context, id, userID uuid.UUID) (*ChatSession, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]ChatSession, error)
	ListByUserAndSource(ctx context.Context, userID uuid.UUID, source string) ([]ChatSession, error)
	Update(ctx context.Context, session *ChatSession) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

type ChatMessageRepository interface {
	Create(ctx context.Context, msg *ChatMessage) error
	ListBySession(ctx context.Context, sessionID, userID uuid.UUID) ([]ChatMessage, error)
	DeleteBySession(ctx context.Context, sessionID, userID uuid.UUID) error
}

type AIProviderRepository interface {
	List(ctx context.Context) ([]AIProvider, error)
	GetByID(ctx context.Context, id uuid.UUID) (*AIProvider, error)
	GetByName(ctx context.Context, name string) (*AIProvider, error)
	Create(ctx context.Context, provider *AIProvider) error
	Update(ctx context.Context, provider *AIProvider) error
}
