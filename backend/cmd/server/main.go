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

	"github.com/fintrack/backend/internal/aigateway/gateway"
	"github.com/fintrack/backend/internal/audit"
	deliveryHTTP "github.com/fintrack/backend/internal/delivery/http"
	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/featureflag"
	infraDB "github.com/fintrack/backend/internal/infrastructure/db"
	"github.com/fintrack/backend/internal/infrastructure/embedding"
	"github.com/fintrack/backend/internal/infrastructure/metrics"
	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
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

	// Redis (cache berversi + pub/sub). Degrades gracefully when unavailable.
	redisCfg := infraRedis.ConfigFromEnv(os.Getenv, func(key string, def int) int {
		if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
			return v
		}
		return def
	})
	redisClient, redisErr := infraRedis.New(bgCtx, redisCfg)
	if redisErr != nil {
		logger.Warn("Redis unavailable — running without cache", zap.Error(redisErr))
	}
	defer redisClient.Close()

	// Install the versioned cache engine (Redis when available, else memory).
	middleware.ConfigureCache(redisClient)

	// Observability: metrics registry (four golden signals + business counters).
	metricsRegistry := metrics.New()

	// Listen for cross-instance cache-invalidation events.
	stopCacheSubscriber := middleware.StartCacheInvalidationSubscriber(bgCtx)

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

	// Audit trail (append-only). Started once the pool is available.
	auditSvc := audit.NewService(audit.NewPostgresRepository(pool), audit.DefaultConfig())
	defer auditSvc.Close()
	_ = auditSvc // consumed by later phases (P03+); wired now so the trail is ready.

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
	aiGatewayRepo := postgres.NewAIGatewayRepo(pool)

	// Centralized AI gateway (P03). Attached to the AI usecase when enabled.
	aiGateway := gateway.New(aiGatewayRepo, gateway.Config{EncryptionKey: encryptionKey})
	healthChecker := gateway.NewHealthChecker(aiGatewayRepo, gateway.HealthConfig{Interval: 60 * time.Second, Timeout: 5 * time.Second})
	healthChecker.Start(bgCtx)
	defer healthChecker.Stop()

	// Feature flag gating the centralized gateway (default OFF for safe rollout).
	flagSvc := featureflag.NewService(featureflag.NewPostgresRepository(pool), infraRedis.NewCacheEngine(redisClient))
	gatewayEnabled := flagSvc.IsEnabled(bgCtx, "feature_ai_gateway_centralized", nil)
	if gatewayEnabled {
		logger.Info("AI gateway centralized: ENABLED")
	} else {
		logger.Info("AI gateway centralized: disabled (feature flag off)")
	}

	// Embedder (RAG). Prefer the gateway embedder when the gateway is enabled,
	// otherwise fall back to the direct Gemini embedder (graceful degradation).
	var embedder embedding.Embedder
	if gatewayEnabled {
		embedder = embedding.NewGatewayEmbedder(aiGateway)
	} else {
		direct := embedding.NewGeminiEmbedder(os.Getenv("GEMINI_EMBEDDING_API_KEY"))
		if !direct.Enabled() {
			logger.Warn("GEMINI_EMBEDDING_API_KEY not set — RAG embeddings disabled")
		}
		embedder = direct
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
	if gatewayEnabled {
		aiUC.WithGateway(aiGateway)
	}
	providerUC := usecase.NewProviderUsecase(providerRepo)
	reportUC := usecase.NewReportUsecase(txRepo)
	authUC := usecase.NewAuthUsecase(userRepo, refreshRepo, []byte(jwtSecret))

	routerConfig := deliveryHTTP.RouterConfig{
		JWTSecret:    jwtSecret,
		CORSOrigin:   corsOrigin,
		CookieSecure: cookieSecure,
		Metrics:      metricsRegistry,
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
	stopCacheSubscriber()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited gracefully")
}
