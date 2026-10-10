package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrDuplicateAlias = errors.New("alias already exists")

type GeminiKeyStatus string

const (
	KeyActive        GeminiKeyStatus = "active"
	KeyQuotaExceeded GeminiKeyStatus = "quota_exceeded"
	KeyError         GeminiKeyStatus = "error"
	KeyInactive      GeminiKeyStatus = "inactive"
)

type GeminiAPIKey struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	UserID        uuid.UUID       `json:"user_id" db:"user_id"`
	Alias         string          `json:"alias" db:"alias"`
	Provider      string          `json:"provider" db:"provider"`
	Model         string          `json:"model,omitempty" db:"model"`
	EncryptedKey  string          `json:"-" db:"encrypted_key"`
	KeyPreview    string          `json:"key_preview" db:"key_preview"`
	Priority      int             `json:"priority" db:"priority"`
	Status        GeminiKeyStatus `json:"status" db:"status"`
	SupportsText  bool            `json:"supports_text" db:"supports_text"`
	SupportsAudio bool            `json:"supports_audio" db:"supports_audio"`
	SupportsVision bool           `json:"supports_vision" db:"supports_vision"`
	LastUsedAt    *time.Time      `json:"last_used_at,omitempty" db:"last_used_at"`
	LastError     *string         `json:"last_error,omitempty" db:"last_error"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

type GeminiKeyResponse struct {
	ID            uuid.UUID       `json:"id"`
	Alias         string          `json:"alias"`
	Provider      string          `json:"provider"`
	Model         string          `json:"model,omitempty"`
	KeyPreview    string          `json:"key_preview"`
	Priority      int             `json:"priority"`
	Status        GeminiKeyStatus `json:"status"`
	SupportsText  bool            `json:"supports_text"`
	SupportsAudio bool            `json:"supports_audio"`
	SupportsVision bool           `json:"supports_vision"`
	LastUsedAt    *time.Time      `json:"last_used_at,omitempty"`
	LastError     *string         `json:"last_error,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

type GeminiKeyRepository interface {
	Create(ctx context.Context, key *GeminiAPIKey) error
	GetByID(ctx context.Context, id, userID uuid.UUID) (*GeminiAPIKey, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]GeminiAPIKey, error)
	Update(ctx context.Context, key *GeminiAPIKey) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
	GetActiveKeysByPriority(ctx context.Context, userID uuid.UUID) ([]GeminiAPIKey, error)
	GetActiveKeysByModality(ctx context.Context, userID uuid.UUID, modality string) ([]GeminiAPIKey, error)
}

func (k *GeminiAPIKey) ToResponse() GeminiKeyResponse {
	return GeminiKeyResponse{
		ID:             k.ID,
		Alias:          k.Alias,
		Provider:       k.Provider,
		Model:          k.Model,
		KeyPreview:     k.KeyPreview,
		Priority:       k.Priority,
		Status:         k.Status,
		SupportsText:   k.SupportsText,
		SupportsAudio:  k.SupportsAudio,
		SupportsVision: k.SupportsVision,
		LastUsedAt:     k.LastUsedAt,
		LastError:      k.LastError,
		CreatedAt:      k.CreatedAt,
	}
}

type CreateGeminiKeyRequest struct {
	Alias         string `json:"alias" binding:"required"`
	APIKey        string `json:"api_key" binding:"required"`
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	Priority      int    `json:"priority"`
	SupportsText  bool   `json:"supports_text"`
	SupportsAudio bool   `json:"supports_audio"`
	SupportsVision bool  `json:"supports_vision"`
}

type UpdateGeminiKeyRequest struct {
	Alias          *string          `json:"alias"`
	APIKey         *string          `json:"api_key"`
	Provider       *string          `json:"provider"`
	Model          *string          `json:"model"`
	Priority       *int             `json:"priority"`
	Status         *GeminiKeyStatus `json:"status"`
	SupportsText   *bool            `json:"supports_text"`
	SupportsAudio  *bool            `json:"supports_audio"`
	SupportsVision *bool            `json:"supports_vision"`
}

// Modality constants for key lookup
const (
	ModalityText   = "text"
	ModalityAudio  = "audio"
	ModalityVision = "vision"
)
