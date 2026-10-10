package main

import (
	"context"
	"encoding/hex"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	deliveryHTTP "github.com/fintrack/backend/internal/delivery/http"
	infraDB "github.com/fintrack/backend/internal/infrastructure/db"
	"github.com/fintrack/backend/internal/infrastructure/embedding"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/repository/postgres"
	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/pkg/logger"
)

func main() {
	_ = godotenv.Load()

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	logger.Init(logLevel)
	logger.Info("Starting FinTrack API server...")

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Fatal("DATABASE_URL is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Fatal("JWT_SECRET is required")
	}

	corsOrigin := os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		corsOrigin = "http://localhost:3000,http://localhost:3001,http://127.0.0.1:3000"
	}

	cookieSecure := os.Getenv("COOKIE_SECURE") == "true"

	encryptionKeyHex := os.Getenv("ENCRYPTION_KEY")
	if encryptionKeyHex == "" {
		logger.Fatal("ENCRYPTION_KEY (32 bytes hex) is required")
	}
	encryptionKey, err := hex.DecodeString(encryptionKeyHex)
	if err != nil {
		logger.Fatal("ENCRYPTION_KEY must be a valid hex string (64 hex chars = 32 bytes)")
	}
	if len(encryptionKey) != 32 {
		logger.Fatal("ENCRYPTION_KEY must decode to exactly 32 bytes (64 hex characters)")
	}

	bgCtx := context.Background()

	poolCfg := infraDB.PoolConfigFromEnv(os.Getenv, func(key string, def int) int {
		if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
			return v
		}
		return def
	})
	pool, err := infraDB.NewPool(bgCtx, databaseURL, poolCfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	// Repositories
	txRepo := postgres.NewTransactionRepo(pool)
	incomeCatRepo := postgres.NewIncomeCategoryRepo(pool)
	expenseCatRepo := postgres.NewExpenseCategoryRepo(pool)
	savingsCatRepo := postgres.NewSavingsCategoryRepo(pool)
	subcatRepo := postgres.NewExpenseSubcategoryRepo(pool)
	userRepo := postgres.NewUserRepo(pool)
	geminiKeyRepo := postgres.NewGeminiKeyRepo(pool)
	spkRepo := postgres.NewSPKRepo(pool)
	chatSessionRepo := postgres.NewChatSessionRepo(pool)
	chatMessageRepo := postgres.NewChatMessageRepo(pool)
	providerRepo := postgres.NewProviderRepo(pool)
	refreshRepo := postgres.NewRefreshTokenRepo(pool)
	embedRepo := postgres.NewEmbeddingRepo(pool)
	txManager := postgres.NewTxManager(pool)

	// Embedder (RAG). Disabled gracefully when the API key is absent.
	embedder := embedding.NewGeminiEmbedder(os.Getenv("GEMINI_EMBEDDING_API_KEY"))
	if !embedder.Enabled() {
		logger.Warn("GEMINI_EMBEDDING_API_KEY not set — RAG embeddings disabled")
	}

	// Category-name resolver used to build embedding text.
	catName := func(ctx context.Context, tx *domain.Transaction) string {
		if tx.IncomeCategoryID != nil {
			if c, err := incomeCatRepo.GetByID(ctx, *tx.IncomeCategoryID); err == nil && c != nil {
				return c.Name
			}
		}
		if tx.ExpenseCategoryID != nil {
			if c, err := expenseCatRepo.GetByID(ctx, *tx.ExpenseCategoryID); err == nil && c != nil {
				return c.Name
			}
		}
		if tx.SavingsCategoryID != nil {
			if c, err := savingsCatRepo.GetByID(ctx, *tx.SavingsCategoryID); err == nil && c != nil {
				return c.Name
			}
		}
		return ""
	}

	// Usecases
	spkUC := usecase.NewSPKUsecase(txRepo, spkRepo, txManager)
	spkQueue := usecase.NewSPKQueue(4, func(ctx context.Context, userID uuid.UUID) {
		if _, err := spkUC.Calculate(ctx, userID); err != nil {
			logger.Warn("SPK calculation failed", zap.Error(err))
		}
	})
	transactionUC := usecase.NewTransactionUsecase(txRepo, spkQueue, embedder, embedRepo, catName)
	categoryUC := usecase.NewCategoryUsecase(incomeCatRepo, expenseCatRepo, savingsCatRepo, subcatRepo)
	userUC := usecase.NewUserUsecase(userRepo)
	dashboardUC := usecase.NewDashboardUsecase(txRepo, spkUC)
	geminiKeyUC := usecase.NewGeminiKeyUsecase(geminiKeyRepo, encryptionKey)
	aiUC := usecase.NewAIUsecase(chatSessionRepo, chatMessageRepo, geminiKeyRepo, txRepo, incomeCatRepo, expenseCatRepo, savingsCatRepo, userRepo, encryptionKey, embedder, embedRepo)
	providerUC := usecase.NewProviderUsecase(providerRepo)
	reportUC := usecase.NewReportUsecase(txRepo)
	authUC := usecase.NewAuthUsecase(userRepo, refreshRepo, []byte(jwtSecret))

	routerConfig := deliveryHTTP.RouterConfig{
		JWTSecret:    jwtSecret,
		CORSOrigin:   corsOrigin,
		CookieSecure: cookieSecure,
	}

	r := deliveryHTTP.SetupRouter(
		routerConfig,
		transactionUC,
		categoryUC,
		userUC,
		dashboardUC,
		geminiKeyUC,
		spkUC,
		aiUC,
		providerUC,
		reportUC,
		authUC,
	)

	// Start the SPK worker pool.
	spkQueue.Start(bgCtx)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("Server listening on :" + port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	spkQueue.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited gracefully")
}
