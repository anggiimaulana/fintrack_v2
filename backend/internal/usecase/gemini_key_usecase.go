package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/infrastructure/crypto"
	"github.com/fintrack/backend/pkg/logger"
)

type GeminiKeyUsecase struct {
	keyRepo    domain.GeminiKeyRepository
	encryptKey []byte
}

func NewGeminiKeyUsecase(keyRepo domain.GeminiKeyRepository, encryptKey []byte) *GeminiKeyUsecase {
	return &GeminiKeyUsecase{
		keyRepo:    keyRepo,
		encryptKey: encryptKey,
	}
}

func (u *GeminiKeyUsecase) Create(ctx context.Context, userID uuid.UUID, req domain.CreateGeminiKeyRequest) (*domain.GeminiAPIKey, error) {
	encrypted, err := crypto.Encrypt([]byte(req.APIKey), u.encryptKey)
	if err != nil {
		logger.Error("failed to encrypt API key", zap.Error(err))
		return nil, fmt.Errorf("failed to encrypt API key: %w", err)
	}

	preview := req.APIKey
	if len(preview) > 8 {
		preview = preview[:8] + "..."
	}

	provider := req.Provider
	if provider == "" {
		provider = "gemini"
	}

	key := &domain.GeminiAPIKey{
		UserID:         userID,
		Alias:          req.Alias,
		Provider:       provider,
		Model:          req.Model,
		EncryptedKey:   encrypted,
		KeyPreview:     preview,
		Priority:       req.Priority,
		Status:         domain.KeyActive,
		SupportsText:   req.SupportsText,
		SupportsAudio:  req.SupportsAudio,
		SupportsVision: req.SupportsVision,
	}

	if err := u.keyRepo.Create(ctx, key); err != nil {
		return nil, err
	}

	return key, nil
}

func (u *GeminiKeyUsecase) List(ctx context.Context, userID uuid.UUID) ([]domain.GeminiAPIKey, error) {
	return u.keyRepo.ListByUser(ctx, userID)
}

func (u *GeminiKeyUsecase) GetByID(ctx context.Context, userID, id uuid.UUID) (*domain.GeminiAPIKey, error) {
	return u.keyRepo.GetByID(ctx, id, userID)
}

func (u *GeminiKeyUsecase) Update(ctx context.Context, userID, id uuid.UUID, req domain.UpdateGeminiKeyRequest) (*domain.GeminiAPIKey, error) {
	key, err := u.keyRepo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, nil
	}

	if req.APIKey != nil && *req.APIKey != "" {
		encrypted, err := crypto.Encrypt([]byte(*req.APIKey), u.encryptKey)
		if err != nil {
			logger.Error("failed to encrypt API key", zap.Error(err))
			return nil, fmt.Errorf("failed to encrypt API key: %w", err)
		}
		key.EncryptedKey = encrypted

		preview := *req.APIKey
		if len(preview) > 8 {
			preview = preview[:8] + "..."
		}
		key.KeyPreview = preview
	}
	if req.Alias != nil {
		key.Alias = *req.Alias
	}
	if req.Provider != nil {
		key.Provider = *req.Provider
	}
	if req.Model != nil {
		key.Model = *req.Model
	}
	if req.Priority != nil {
		key.Priority = *req.Priority
	}
	if req.Status != nil {
		key.Status = *req.Status
	}
	if req.SupportsText != nil {
		key.SupportsText = *req.SupportsText
	}
	if req.SupportsAudio != nil {
		key.SupportsAudio = *req.SupportsAudio
	}
	if req.SupportsVision != nil {
		key.SupportsVision = *req.SupportsVision
	}

	if err := u.keyRepo.Update(ctx, key); err != nil {
		return nil, err
	}

	return key, nil
}

func (u *GeminiKeyUsecase) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return u.keyRepo.Delete(ctx, id, userID)
}

func (u *GeminiKeyUsecase) Test(ctx context.Context, userID, id uuid.UUID) (map[string]interface{}, error) {
	key, err := u.keyRepo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, fmt.Errorf("API key not found")
	}

	plaintext, err := crypto.Decrypt(key.EncryptedKey, u.encryptKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt API key: %w", err)
	}

	return map[string]interface{}{
		"key_preview": key.KeyPreview,
		"status":      "test_passed",
		"message":     "API key decrypted successfully. Actual Gemini API test is handled by the AI service.",
		"key_length":  len(plaintext),
	}, nil
}
