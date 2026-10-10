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
	ID        uuid.UUID `json:"id" db:"id"`
	SessionID uuid.UUID `json:"session_id" db:"session_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Role      string    `json:"role" db:"role"`
	Content   string    `json:"content" db:"content"`
	ModelUsed *string   `json:"model_used,omitempty" db:"model_used"`
	TokensUsed *int     `json:"tokens_used,omitempty" db:"tokens_used"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type AIProvider struct {
	ID              uuid.UUID `json:"id" db:"id"`
	Name            string    `json:"name" db:"name"`
	DisplayName     string    `json:"display_name" db:"display_name"`
	SupportsFast    bool      `json:"supports_fast" db:"supports_fast"`
	SupportsReasoning bool    `json:"supports_reasoning" db:"supports_reasoning"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

type CreateProviderRequest struct {
	Name            string `json:"name" binding:"required"`
	DisplayName     string `json:"display_name" binding:"required"`
	SupportsFast    bool   `json:"supports_fast"`
	SupportsReasoning bool `json:"supports_reasoning"`
	IsActive        bool   `json:"is_active"`
}

type UpdateProviderRequest struct {
	Name            *string `json:"name"`
	DisplayName     *string `json:"display_name"`
	SupportsFast    *bool   `json:"supports_fast"`
	SupportsReasoning *bool `json:"supports_reasoning"`
	IsActive        *bool   `json:"is_active"`
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
	Type        string                 `json:"type"`
	Content     string                 `json:"content,omitempty"`
	Step        string                 `json:"step,omitempty"`
	StepLabel   string                 `json:"step_label,omitempty"`
	StepStatus  string                 `json:"step_status,omitempty"`
	Done        bool                   `json:"done,omitempty"`
	ToolCall    *ToolCallInfo          `json:"tool_call,omitempty"`
	ToolResult  *ToolResultInfo        `json:"tool_result,omitempty"`
}

type ChatSessionRepository interface {
	Create(ctx context.Context, session *ChatSession) error
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
