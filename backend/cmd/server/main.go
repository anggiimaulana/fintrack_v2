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

	"github.com/fintrack/backend/internal/admin"
	"github.com/fintrack/backend/internal/aigateway/bootstrap"
	"github.com/fintrack/backend/internal/aigateway/breaker"
	"github.com/fintrack/backend/internal/aigateway/gateway"
	"github.com/fintrack/backend/internal/aigateway/metering"
	"github.com/fintrack/backend/internal/aituning"
	"github.com/fintrack/backend/internal/audit"
	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/internal/configcatalog"
	deliveryHTTP "github.com/fintrack/backend/internal/delivery/http"
	"github.com/fintrack/backend/internal/delivery/http/handler"
	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/entitlement"
	"github.com/fintrack/backend/internal/featureflag"
	infraDB "github.com/fintrack/backend/internal/infrastructure/db"
	"github.com/fintrack/backend/internal/infrastructure/embedding"
	"github.com/fintrack/backend/internal/infrastructure/metrics"
	infraRedis "github.com/fintrack/backend/internal/infrastructure/redis"
	"github.com/fintrack/backend/internal/mail"
	"github.com/fintrack/backend/internal/notification"
	"github.com/fintrack/backend/internal/payment"
	"github.com/fintrack/backend/internal/realtime"
	"github.com/fintrack/backend/internal/repository/postgres"
	"github.com/fintrack/backend/internal/runtimeconfig"
	"github.com/fintrack/backend/internal/storage"
	"github.com/fintrack/backend/internal/voice"
	"github.com/fintrack/backend/internal/usage"
	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/pkg/contracts"
	"github.com/fintrack/backend/pkg/logger"
)

// shutdownTimeout is the graceful-drain budget (default 30s), overridable via
// SHUTDOWN_TIMEOUT in seconds.
func shutdownTimeout() time.Duration {
	if v := os.Getenv("SHUTDOWN_TIMEOUT"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return 30 * time.Second
}

// envSeconds reads a duration in seconds from an env var, falling back to def
// when the value is missing or not a positive integer.
func envSeconds(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return def
}

// startPricingRefresh keeps the pricing engine's active rule in sync with the
// database. It refreshes periodically so a Super Admin change (or an out-of-band
// edit) is picked up without a restart, and returns a stop function.
//
// The refresh is cheap (one indexed row) and best-effort: a transient DB error
// leaves the previous rule in place rather than zeroing costs.
func startPricingRefresh(ctx context.Context, repo *postgres.BillingRepo, engine *billing.PricingEngine) func() {
	ticker := time.NewTicker(1 * time.Minute)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				rule, err := repo.GetActiveRule(ctx)
				if err != nil {
					logger.Warn("pricing rule refresh failed", zap.Error(err))
					continue
				}
				engine.SetRule(rule)
			}
		}
	}()
	return func() {
		ticker.Stop()
		<-done
	}
}

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
	_ = auditSvc // consumed by the Super Admin audit viewer (P06-017).

	// Repositories
	txRepo := postgres.NewTransactionRepo(pool)
	incomeCatRepo := postgres.NewIncomeCategoryRepo(pool)
	expenseCatRepo := postgres.NewExpenseCategoryRepo(pool)
	savingsCatRepo := postgres.NewSavingsCategoryRepo(pool)
	subcatRepo := postgres.NewExpenseSubcategoryRepo(pool)
	userRepo := postgres.NewUserRepo(pool)
	chatSessionRepo := postgres.NewChatSessionRepo(pool)
	chatMessageRepo := postgres.NewChatMessageRepo(pool)
	providerRepo := postgres.NewProviderRepo(pool)
	refreshRepo := postgres.NewRefreshTokenRepo(pool)
	embedRepo := postgres.NewEmbeddingRepo(pool)
	txManager := postgres.NewTxManager(pool)
	aiGatewayRepo := postgres.NewAIGatewayRepo(pool)

	// Emergency kill switch (P06-016): persisted in `settings`, fanned out over
	// Pub/Sub so every instance stops routing a use case within ~100ms.
	killSwitch := admin.NewKillSwitchStore(pool, infraRedis.NewPubSub(redisClient))
	if err := killSwitch.Load(bgCtx); err != nil {
		logger.Warn("AI kill switch state not loaded (using defaults)", zap.Error(err))
	}
	stopKillSwitchWatch := killSwitch.Watch(bgCtx)
	defer stopKillSwitchWatch()

	// Centralized AI gateway (P03). Attached to the AI usecase when enabled.
	// Realtime SSE (P07 slice B): hub for local streams, Redis pub/sub for
	// cross-instance fan-out, ring-buffer for Last-Event-ID resume.
	realtimeHub := realtime.NewHub()
	realtimePubSub := infraRedis.NewPubSub(redisClient)
	realtimeBuffer := realtime.NewBuffer(redisClient)
	realtimePublisher := realtime.NewPublisher(realtimePubSub, realtimeHub, realtimeBuffer)
	realtimeHandler := realtime.NewHandler(realtimeHub, realtimeBuffer)
	stopRealtimeSub := realtime.NewSubscriber(realtimePubSub, realtimeHub).Start(bgCtx)
	defer stopRealtimeSub()
	stopSweeper := realtimeHub.StartIdleSweeper(bgCtx, 2*time.Minute, 1*time.Minute)
	defer stopSweeper()
	defer realtimeHub.CloseAll()

	var aiGateway *gateway.Service
	aiGateway = gateway.New(aiGatewayRepo, gateway.Config{
		EncryptionKey: encryptionKey,
		// DefaultTimeout bounds a single upstream call when a route does not set
		// its own timeout_ms. Vision/receipt and voice calls routinely exceed the
		// old 30s ceiling on a loaded provider, so the default is raised and can
		// be tuned per-deployment with AI_GATEWAY_TIMEOUT (seconds). A route's
		// timeout_ms (Super Admin → AI Gateway) still overrides this.
		DefaultTimeout: envSeconds("AI_GATEWAY_TIMEOUT", 90*time.Second),
		KillSwitch:     func(uc domain.UseCase) bool { return killSwitch.Enabled(string(uc)) },
		Breaker: breaker.Config{
			OnStateChange: func(endpointID string, from, to breaker.State) {
				if to != breaker.StateOpen {
					return
				}
				realtimePublisher.PublishAdmins(bgCtx, contracts.NewEvent(
					contracts.EventAIEndpointDown, time.Now().Unix(),
					contracts.AIEndpointDownPayload{
						EndpointID:   endpointID,
						FailureCount: aiGateway.Breaker().Failures(endpointID),
						Status:       string(to),
					}))
			},
		},
		// Smart failover: alert admins in realtime when a single route (model) is
		// quarantined, and again when it recovers. This is model-level, distinct
		// from the endpoint-level EventAIEndpointDown above.
		OnRouteStateChange: func(ev gateway.RouteStateChange) {
			realtimePublisher.PublishAdmins(bgCtx, contracts.NewEvent(
				contracts.EventAIRouteQuarantined, time.Now().Unix(),
				contracts.AIRouteQuarantinedPayload{
					UseCase:    ev.UseCase,
					Model:      ev.Model,
					Endpoint:   ev.Endpoint,
					EndpointID: ev.EndpointID,
					Combo:      ev.Combo,
					Reason:     ev.Reason,
					Recovered:  !ev.Opened,
				}))
		},
	})

	// Seed the AI gateway configuration from environment defaults the FIRST time
	// the platform runs against an empty database. Once the Super Admin has
	// configured endpoints/routes/flags through the admin UI, the database is
	// authoritative and the env defaults are ignored for that table.
	//
	// This runs BEFORE the health checker starts so a freshly seeded endpoint is
	// probed on the first tick and the gateway can come up in the same boot.
	bootRes, bootErr := bootstrap.EnsureSeeded(bgCtx, pool, encryptionKey, bootstrap.ConfigFromEnv(os.Getenv))
	if bootErr != nil {
		logger.Warn("AI gateway bootstrap failed", zap.Error(bootErr))
	} else if bootRes.EndpointSeeded || bootRes.RoutesSeeded > 0 || bootRes.FlagSeeded {
		logger.Info("AI gateway bootstrap: seeded from environment defaults",
			zap.Bool("endpoint", bootRes.EndpointSeeded),
			zap.Int("routes", bootRes.RoutesSeeded),
			zap.Bool("flag", bootRes.FlagSeeded))
	}

	healthChecker := gateway.NewHealthChecker(aiGatewayRepo, gateway.HealthConfig{Interval: 60 * time.Second, Timeout: 5 * time.Second})
	// Probe synchronously first: the enablement decision below must not read a
	// stale "unknown" health for an endpoint we just seeded.
	healthChecker.RunOnce(bgCtx)
	healthChecker.Start(bgCtx)
	defer healthChecker.Stop()

	// One cache engine shared process-wide: version bumps (role demotion,
	// entitlement invalidation) must be visible to every consumer, which the
	// in-memory backend cannot guarantee across separate instances.
	cacheEngine := infraRedis.NewCacheEngine(redisClient)

	// Feature flag gating the centralized gateway (default OFF for safe rollout).
	flagSvc := featureflag.NewService(featureflag.NewPostgresRepository(pool), cacheEngine)
	// The centralized gateway is an architectural switch, not a per-user rollout,
	// so it is read with IsEnabledForPlatform (IsEnabled(…, nil) returns false for
	// any partial rollout — which is how this gateway silently stayed off).
	//
	// Safety guard: switching the platform onto the gateway with no routes
	// configured would send every AI request into a dead end. So the flag alone is
	// not sufficient — at least one active chat route must exist.
	gatewayEnabled := flagSvc.IsEnabledForPlatform(bgCtx, "feature_ai_gateway_centralized")
	if gatewayEnabled {
		// A route alone is not sufficient: the gateway is the only AI transport
		// (the legacy BYOK path was removed), so pointing the platform at an
		// endpoint whose health check is failing would break AI for every user.
		// Both a route AND a healthy endpoint are required. "unknown" health does
		// not count — an unprobed endpoint is not evidence that it can serve traffic.
		healthy, err := aiGatewayRepo.HasHealthyEndpoint(bgCtx, domain.UseCaseChat)
		if err != nil {
			logger.Warn("AI gateway: endpoint health check failed; keeping gateway disabled", zap.Error(err))
			healthy = false
		}
		if !healthy {
			hasRoutes, rerr := aiGatewayRepo.HasActiveRoutes(bgCtx, domain.UseCaseChat)
			if rerr == nil && hasRoutes {
				logger.Warn("AI gateway: flag is ON and a chat route exists, but no endpoint is healthy; " +
					"gateway stays disabled (AI unavailable)")
			} else {
				logger.Warn("AI gateway: flag is ON but no active chat route is configured; " +
					"gateway stays disabled (AI unavailable)")
			}
		}
		gatewayEnabled = healthy
	}
	if gatewayEnabled {
		logger.Info("AI gateway centralized: ENABLED")

		// Verify the ACTUAL model on every active route answers — not just that
		// the endpoint is reachable. An endpoint can be "healthy" (it responds)
		// while the model its route points at is dead (e.g. the gateway answers
		// 503 for an unknown model). Reachability probing cannot catch that; only
		// a real call can. This runs in the background so a slow or dead model
		// never blocks startup, and it is read-only: it never changes routes,
		// health, or the breaker, so a false negative cannot take AI down.
		go func() {
			results := aiGateway.VerifyRoutes(bgCtx)
			var broken []string
			for _, r := range results {
				switch {
				case r.Skipped:
					logger.Info("AI route verify: skipped (no synthetic fixture)",
						zap.String("use_case", r.UseCase), zap.String("model", r.Model))
				case r.OK:
					logger.Info("AI route verify: OK",
						zap.String("use_case", r.UseCase), zap.String("model", r.Model),
						zap.Int64("latency_ms", r.LatencyMS))
				default:
					broken = append(broken, r.UseCase+"/"+r.Model)
					logger.Warn("AI route verify: MODEL NOT RESPONDING — this route will fail for users",
						zap.String("use_case", r.UseCase),
						zap.String("model", r.Model),
						zap.String("endpoint", r.Endpoint),
						zap.String("error", r.Error))
				}
			}
			if len(broken) > 0 {
				// Smart failover: quarantine the dead routes so the very next
				// request fails over to a healthy sibling instead of paying the
				// dead-model round-trip. Recovery is automatic (breaker cooldown),
				// so a model that comes back is used again without a restart.
				quarantined := aiGateway.QuarantineBrokenRoutes(results)
				logger.Error("AI route verify: broken routes detected; quarantined for fast failover",
					zap.Strings("routes", broken),
					zap.Int("quarantined", quarantined))
			}
		}()
	} else {
		logger.Info("AI gateway centralized: disabled")
	}

	// Central runtime configuration: DB setting wins, env supplies the default.
	// Used by payment credentials, R2, mail and AI knobs so a Super Admin change
	// applies without a redeploy. Declared early because the embedder, gateways
	// and storage provider all resolve through it.
	settingsRepo := postgres.NewSettingsRepo(pool)
	runtimeResolver := runtimeconfig.New(settingsRepo, encryptionKey)

	// Embedder (RAG). Prefer the gateway embedder when the gateway is enabled,
	// otherwise fall back to the direct Gemini embedder (graceful degradation).
	var embedder embedding.Embedder
	if gatewayEnabled {
		embedder = embedding.NewGatewayEmbedder(aiGateway)
	} else {
		direct := embedding.NewGeminiEmbedder(
			runtimeResolver.Secret(bgCtx, "gemini_embedding_api_key", "GEMINI_EMBEDDING_API_KEY"))
		if !direct.Enabled() {
			logger.Warn("GEMINI_EMBEDDING_API_KEY not set — RAG embeddings disabled")
		}
		embedder = direct
	}
	// The emergency stop covers embeddings on every transport: the gateway
	// embedder consults it internally, the direct one is wrapped here.
	embedder = embedding.NewKillSwitchEmbedder(embedder, func() bool {
		return killSwitch.Enabled(string(domain.UseCaseEmbedding))
	})

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
	spkRepo := postgres.NewSPKRepo(pool)
	spkMetricRepo := postgres.NewSPKMetricRepo(pool)
	spkMetricUC := usecase.NewSPKMetricUsecase(spkMetricRepo)
	budgetRepo := postgres.NewBudgetRepo(pool)
	budgetUC := usecase.NewBudgetUsecase(budgetRepo, txRepo)
	paymentAccountRepo := postgres.NewPaymentAccountRepo(pool)
	paymentAccountUC := usecase.NewPaymentAccountUsecase(paymentAccountRepo)
	spkUC := usecase.NewSPKUsecase(txRepo, spkRepo, spkMetricRepo, txManager)
	spkQueue := usecase.NewSPKQueue(4, func(ctx context.Context, userID uuid.UUID) {
		if _, err := spkUC.Calculate(ctx, userID); err != nil {
			logger.Warn("SPK calculation failed", zap.Error(err))
		}
	})
	transactionUC := usecase.NewTransactionUsecase(txRepo, spkQueue, embedder, embedRepo, catName)
	categoryUC := usecase.NewCategoryUsecase(incomeCatRepo, expenseCatRepo, savingsCatRepo, subcatRepo)
	userUC := usecase.NewUserUsecase(userRepo)
	dashboardUC := usecase.NewDashboardUsecase(txRepo, spkUC)

	aiUC := usecase.NewAIUsecase(chatSessionRepo, chatMessageRepo, txRepo, incomeCatRepo, expenseCatRepo, savingsCatRepo, userRepo, embedder, embedRepo)
	if gatewayEnabled {
		aiUC.WithGateway(aiGateway)
	} else {
		// The centralized gateway is the only transport that can actually serve
		// chat: per-user BYOK keys were removed (migration 0014 dropped
		// gemini_api_keys), so with the gateway off every chat request ends in the
		// degraded "unreachable" message. Surface this loudly rather than letting
		// AI appear silently broken.
		logger.Warn("AI gateway centralized: DISABLED — chat has no working transport " +
			"(BYOK keys removed); enable feature_ai_gateway_centralized with a healthy chat route")
	}
	// The emergency stop applies regardless of which transport serves a model
	// call, so it is wired outside the gateway feature flag.
	aiUC.WithKillSwitch(killSwitch.Enabled)
	// `reasoning_effort` is resolved at request time (DB setting over env) so the
	// Super Admin can tune it without a redeploy.
	aiUC.WithReasoningEffort(func() string {
		return runtimeResolver.String(bgCtx, "ai_reasoning_effort", "AI_REASONING_EFFORT", "")
	})
	// Subscription & entitlement engine (P04).
	billingRepo := postgres.NewBillingRepo(pool)
	entSource := postgres.NewEntitlementSource(billingRepo)
	entResolver := entitlement.NewResolver(entSource, cacheEngine)
	aiUC.WithEntitlements(entResolver)
	entHandler := handler.NewEntitlementHandler(entResolver)
	// User usage: a chart time series + a paginated consumption ledger, derived
	// from token_ledger and transactions (no new producer required).
	usageRepo := postgres.NewUsageRepo(pool)
	usageHandler := handler.NewUsageHandler(usage.NewService(usageRepo), usageRepo)

	// Voice input: durable per-user transcription history, the dynamic voice
	// config, and the retention worker. The config reader resolves DB-over-env so
	// every knob is tunable from the Super Admin UI (K-VOI-19).
	voiceRepo := postgres.NewVoiceTranscriptionRepo(pool)
	voiceHandler := handler.NewVoiceHandler(voiceRepo, entResolver, runtimeResolver)
	voiceRetention := voice.NewRetentionWorker(voiceRepo, func(ctx context.Context) int {
		return voice.Load(ctx, runtimeResolver).HistoryRetentionDay
	}, voice.RetentionConfig{})
	voiceRetention.Start(bgCtx)
	defer voiceRetention.Stop()

	entBroadcaster := entitlement.NewBroadcaster(infraRedis.NewPubSub(redisClient), entResolver)
	stopEntSubscriber := entBroadcaster.StartSubscriber(bgCtx)
	defer stopEntSubscriber()

	// Payments & billing (P05).
	paymentRepo := postgres.NewPaymentRepo(pool)

	// Apply any Super-Admin-tuned breaker thresholds to the running gateway
	// (created earlier with built-in defaults). A no-op when unset.
	applyBreakerSettings(bgCtx, aiGateway, settingsRepo)

	// Apply any Super-Admin-tuned token metering heuristics (local estimator).
	applyMeteringSettings(bgCtx, settingsRepo)

	// AI timeout budget: apply the Super-Admin-configured per-use-case budgets to
	// the running gateway (and the HTTP write-deadline) at boot, then keep every
	// pod in sync via Pub/Sub so a change applies fleet-wide without a restart.
	// The applier re-reads the database on each notification, so a missed message
	// self-heals on the next change.
	settingsPubSub := infraRedis.NewPubSub(redisClient)
	aiTuner := aituning.New(aiGateway, runtimeResolver)
	aiTuner.Apply(bgCtx)
	stopSettingsWatch := settingsPubSub.Subscribe(bgCtx, infraRedis.ChannelSettingsChanged, func(group string) {
		// Drop the resolver's short-lived cache FIRST: otherwise the apply below
		// would re-read the stale value on this pod and the change would only
		// land after the TTL — defeating the point of the broadcast.
		if runtimeResolver != nil {
			runtimeResolver.Invalidate()
		}
		// Only the AI group touches the gateway/HTTP budgets; other settings
		// changes (mail, payment, …) are handled by their own resolvers.
		if group == "ai" || group == "settings" {
			aiTuner.Apply(bgCtx)
		}
	})
	defer stopSettingsWatch()

	// Notifications: durable in-app inbox + email that honours per-user
	// preferences. Mail config resolves DB setting -> env default, so an operator
	// can change SMTP without a redeploy while env still works out of the box.
	notificationRepo := postgres.NewNotificationRepo(pool)
	mailProvider := mail.NewProvider(settingsRepo, encryptionKey)
	notifService := notification.NewService(notificationRepo, mailProvider,
		func(ctx context.Context, uid uuid.UUID) (string, error) {
			p, err := userRepo.GetByID(ctx, uid)
			if err != nil || p == nil {
				return "", err
			}
			return p.Email, nil
		}).WithEmitter(realtimePublisher)
	notifHandler := handler.NewNotificationHandler(notificationRepo, notifService)
	// Wrap the realtime publisher so every user-facing event is also written to
	// the durable inbox (and emailed when permitted). Payment/billing keep
	// emitting to the same interface, unaware of persistence.
	notifEmitter := notification.NewEmitter(realtimePublisher, notifService)
	payRegistry := payment.NewRegistry()
	// Gateways resolve their credentials at call time (DB setting over env), so
	// Midtrans keys and bank details can change without a redeploy.
	payRegistry.Register(payment.NewDynamicMidtransGateway(runtimeResolver))
	payRegistry.Register(payment.NewDynamicManualGateway(runtimeResolver))
	payTax := payment.NewTaxCalculator(settingsRepo)
	payInvoice := payment.NewInvoiceService(paymentRepo, settingsRepo, payTax)
	subActivator := usecase.NewSubscriptionActivator(billingRepo, billingRepo, entResolver)
	payWebhook := payment.NewWebhookProcessor(paymentRepo, payRegistry, subActivator, txManager)

	// Proof object storage (P0). Local by default; R2 when configured. The
	// active driver is read from the `storage_driver` setting at runtime, so an
	// admin can switch local <-> r2 without a restart. R2 credentials resolve at
	// runtime too, so a Super Admin change applies without a redeploy.
	localStore, err := storage.NewLocalStore(os.Getenv("STORAGE_LOCAL_DIR"))
	if err != nil {
		logger.Fatal("Failed to initialise local storage", zap.Error(err))
	}
	r2Resolver := storage.NewDynamicR2Resolver(runtimeResolver)
	if store, rerr := r2Resolver.R2(bgCtx); rerr == nil && store != nil {
		logger.Info("R2 storage configured", zap.String("bucket", store.Bucket()))
	} else {
		logger.Info("R2 storage not configured (using local storage)")
	}
	defaultDriver := runtimeResolver.String(bgCtx, "storage_driver", "STORAGE_DRIVER", storage.DriverLocal)
	if defaultDriver == "" {
		defaultDriver = storage.DriverLocal
	}
	storageProvider := storage.NewProvider(storage.Config{
		DefaultDriver: defaultDriver,
		Local:         localStore,
		R2Resolver:    r2Resolver,
		Settings:      settingsRepo,
	})
	proofStore := storage.NewProofStore(storageProvider)

	payManual := payment.NewManualService(paymentRepo, subActivator, txManager, proofStore)
	payAdmin := payment.NewAdminService(payManual)
	payCheckout := payment.NewCheckoutService(paymentRepo, settingsRepo, payRegistry, payInvoice, nil).WithTx(txManager)
	payUserSvc := payment.NewUserService(paymentRepo, settingsRepo, payRegistry, payManual).
		WithPaymentAccounts(payment.NewPaymentAccountAdapter(paymentAccountRepo))
	// The public plan catalog annotates each limit with its label/unit and —
	// importantly — whether the backend enforces it, so the user-facing pricing
	// UI never advertises a limit that is not actually enforced (F4).
	catalogLimitMeta := make([]payment.LimitMeta, 0, len(entitlement.LimitCatalog))
	for _, lk := range entitlement.LimitCatalog {
		catalogLimitMeta = append(catalogLimitMeta, payment.LimitMeta{
			Key: lk.Key, Label: lk.Label, Unit: lk.Unit, Enforced: lk.Enforced,
		})
	}
	payCatalog := payment.NewCatalogService(billingRepo, billingRepo, catalogLimitMeta, payTax)
	paySubSvc := payment.NewSubscriptionService(billingRepo, billingRepo, nil, func(ctx context.Context, uid uuid.UUID) {
		entBroadcaster.PublishInvalidate(ctx, uid)
	})
	payHandler := handler.NewPaymentHandler(payWebhook).
		WithManual(payManual).
		WithAdmin(payAdmin).
		WithCheckout(payCheckout, billingRepo, userRepo).
		WithUserService(payUserSvc).
		WithCatalog(payCatalog).
		WithSubscriptionService(paySubSvc, billingRepo).
		WithStorage(proofStore)

	// Realtime producers (P07 slice B).
	payWebhook.SetEmitter(notifEmitter)
	payManual.SetEmitter(notifEmitter)

	// Dunning & grace workers. The grace worker needs a status setter so an
	// overdue subscription actually transitions to `grace`/`past_due`; without it
	// the scan ran but nothing changed (F3).
	setSubStatus := func(ctx context.Context, userIDStr string, status string) error {
		uid, err := uuid.Parse(userIDStr)
		if err != nil {
			return err
		}
		return billingRepo.SetSubscriptionStatus(ctx, uid, billing.SubscriptionStatus(status))
	}
	dunningWorker := payment.NewDunningWorker(paymentRepo, settingsRepo, nil, time.Hour)
	graceWorker := payment.NewGraceWorker(paymentRepo, settingsRepo, setSubStatus, 30*time.Minute)
	// Auto-expire unpaid invoices past their due date (dynamic window set by the
	// operator as hours + minutes). A 5-minute scan keeps the lag small.
	expiryWorker := payment.NewExpiryWorker(paymentRepo, 5*time.Minute)
	dunningWorker.Start(bgCtx)
	graceWorker.Start(bgCtx)
	expiryWorker.Start(bgCtx)
	defer dunningWorker.Stop()
	defer graceWorker.Stop()
	defer expiryWorker.Stop()

	providerUC := usecase.NewProviderUsecase(providerRepo)
	reportUC := usecase.NewReportUsecase(txRepo)
	authUC := usecase.NewAuthUsecase(userRepo, refreshRepo, []byte(jwtSecret))

	// Entitlement enforcement (P07 slice A): the middleware existed since P04 but
	// was never constructed, so plan checks never ran. Wire it now, with usage
	// resolvers backed by the transaction and token-period repositories.
	entMiddleware := middleware.NewEntitlementMiddleware(entResolver)
	// Config catalog (P-13): the limit→resolver mapping, setting validators, and
	// feature/metric catalogs are DATA, not code. Version-stamped so a Super
	// Admin change is visible on the next read without a deploy or a TTL wait.
	catalogStore := configcatalog.NewCatalogStore(pool, cacheEngine)
	entUsage := &middleware.UsageProviders{
		Catalog:           catalogStore,
		CountTransactions: txRepo.CountActiveByUser,
		CurrentTokens:     middleware.TokenPeriodUsage(billingRepo),
		CountAIRequests: func(ctx context.Context, userID uuid.UUID) (int64, error) {
			return billingRepo.CountAIRequestsInPeriod(ctx, userID, time.Now().UTC().Unix())
		},
		CountVisionScans: func(ctx context.Context, userID uuid.UUID) (int64, error) {
			return billingRepo.CountVisionScansInPeriod(ctx, userID, time.Now().UTC().Unix())
		},
		CountVoiceToday: func(ctx context.Context, userID uuid.UUID) (int64, error) {
			day, _ := voice.DayWindow(time.Now().UTC(), voice.Load(ctx, runtimeResolver).ResetTimezone)
			return voiceRepo.DailyUsage(ctx, userID, day)
		},
	}

	// F6: expose real-time usage on GET /me/entitlements. The counters mirror the
	// middleware's resolvers so the UI shows exactly what the gate enforces.
	entHandler.WithUsage(&handler.UsageProvidersFn{
		Counters: map[string]func(ctx context.Context, userID uuid.UUID) (int64, error){
			"ai_tokens_per_cycle": middleware.TokenPeriodUsage(billingRepo),
			"max_transactions":    txRepo.CountActiveByUser,
			"ocr_scans_per_cycle": func(ctx context.Context, userID uuid.UUID) (int64, error) {
				return billingRepo.CountVisionScansInPeriod(ctx, userID, time.Now().UTC().Unix())
			},
			"ai_requests_per_cycle": func(ctx context.Context, userID uuid.UUID) (int64, error) {
				return billingRepo.CountAIRequestsInPeriod(ctx, userID, time.Now().UTC().Unix())
			},
			"voice_requests_per_day": func(ctx context.Context, userID uuid.UUID) (int64, error) {
				day, _ := voice.DayWindow(time.Now().UTC(), voice.Load(ctx, runtimeResolver).ResetTimezone)
				return voiceRepo.DailyUsage(ctx, userID, day)
			},
		},
		UnitFor: func(ctx context.Context, limitKey string) string {
			if m, ok := catalogStore.Metric(ctx, limitKey); ok {
				return m.Unit
			}
			return ""
		},
		Period: func(ctx context.Context, userID uuid.UUID) (*entitlement.Period, error) {
			p, err := billingRepo.GetCurrentPeriod(ctx, userID, time.Now().UTC().Unix())
			if err != nil || p == nil {
				return nil, err
			}
			return &entitlement.Period{
				Start: p.PeriodStart.Format(time.RFC3339),
				End:   p.PeriodEnd.Format(time.RFC3339),
			}, nil
		},
	})

	// Settings-backed rounding policy + dynamic sentinel. Both read from the
	// settings store so an operator can change them without a deploy.
	quotaCfg := billing.NewSettingsQuotaConfig(settingsRepo)

	quotaResolver := billing.QuotaResolverFunc(func(ctx context.Context, userID uuid.UUID, at time.Time) (time.Time, time.Time, int64, int64, error) {
		res, err := entResolver.Resolve(ctx, userID)
		if err != nil {
			return time.Time{}, time.Time{}, 0, 0, err
		}
		quota := res.Limit("ai_tokens_per_cycle")
		if quota < 0 {
			// Dynamic sentinel (setting quota_unlimited_sentinel), not a constant.
			quota = quotaCfg.UnlimitedSentinel(ctx)
		}
		ratio := res.QuotaRatioValue()
		if ratio < 1 {
			ratio = 1
		}
		sub, _ := billingRepo.GetByUserID(ctx, userID)
		if sub != nil && sub.CurrentPeriodEnd != nil {
			return sub.CurrentPeriodStart, *sub.CurrentPeriodEnd, quota, ratio, nil
		}
		start := at.Truncate(24 * time.Hour)
		return start, start.AddDate(0, 0, 30), quota, ratio, nil
	})
	// Pricing engine (F2): converts committed tokens to integer Rupiah. The
	// active rule lives in `pricing_rules`; the engine starts from the current
	// rule and is refreshed on demand (admin mutation) and periodically, so a
	// Super Admin change takes effect without a restart.
	pricingEngine := billing.NewPricingEngine(nil)
	if rule, perr := billingRepo.GetActiveRule(bgCtx); perr != nil {
		logger.Warn("No active pricing rule; AI cost will be recorded as 0", zap.Error(perr))
	} else {
		pricingEngine.SetRule(rule)
	}
	ledgerService := billing.NewLedgerService(billingRepo, txManager, quotaResolver, pricingEngine).WithRounding(quotaCfg)
	stopPricingRefresh := startPricingRefresh(bgCtx, billingRepo, pricingEngine)
	defer stopPricingRefresh()

	// Ledger cleaner (F3): auto-releases reservations that were never committed
	// or released (e.g. the process crashed mid-request), so quota does not stay
	// locked forever.
	ledgerCleaner := billing.NewCleanerWorker(billingRepo, ledgerService, billing.CleanerConfig{})
	ledgerCleaner.Start(bgCtx)
	defer ledgerCleaner.Stop()

	// Scheduled downgrades (F3): applies end-of-cycle downgrades written by
	// ScheduleDowngrade. Without this worker the schedule was recorded but never
	// applied, so users kept the higher plan indefinitely.
	subManager := billing.NewSubscriptionManager(billingRepo, billingRepo, nil, txManager)
	downgradeWorker := billing.NewDowngradeWorker(billingRepo, func(ctx context.Context, uid uuid.UUID) error {
		if err := subManager.ApplyScheduledDowngrade(ctx, uid); err != nil {
			return err
		}
		// The plan changed, so the entitlement cache must be invalidated.
		entBroadcaster.PublishInvalidate(ctx, uid)
		return nil
	}, billing.DowngradeConfig{})
	downgradeWorker.Start(bgCtx)
	defer downgradeWorker.Stop()

	// Wrap the realtime publisher so every user-facing event is also written to
	// the durable inbox (and emailed when permitted). Payment/billing keep
	// emitting to the same interface, unaware of persistence.
	routerConfig := deliveryHTTP.RouterConfig{
		JWTSecret:     jwtSecret,
		CORSOrigin:    corsOrigin,
		CookieSecure:  cookieSecure,
		Metrics:       metricsRegistry,
		Entitlement:   entMiddleware,
		Usage:         entUsage,
		Pool:          pool,
		Redis:         redisClient,
		Storage:      storageProvider,
		Ledger:       ledgerService,
		Realtime:     realtimePublisher,
		Audit:        auditSvc,
		Estimates:     billing.NewEstimateStore(settingsRepo),
		Notifications: notifHandler,
		Voice:         voiceHandler,
		VoiceRepo:     voiceRepo,
		VoiceConfig:   runtimeResolver,
		VoiceDaily: func(ctx context.Context, userID uuid.UUID, limit int64) (bool, int64, error) {
			day, _ := voice.DayWindow(time.Now().UTC(), voice.Load(ctx, runtimeResolver).ResetTimezone)
			return voiceRepo.ConsumeDailyUsage(ctx, userID, day, limit)
		},
		EntitlementResolver: entResolver,
		UsageHandler:  usageHandler,
		AIConfig:      handler.NewAIConfigHandler(runtimeResolver),
	}

	// Super Admin portal (P06): one service + handler shared by every route.
	adminSvc := admin.NewService(pool, auditSvc,
		admin.WithManualService(payManual),
		admin.WithGateway(aiGateway),
		admin.WithKillSwitch(killSwitch),
		admin.WithFeatureFlags(flagSvc),
		admin.WithEncryptionKey(encryptionKey),
		admin.WithCache(cacheEngine),
		admin.WithEntitlementInvalidator(entBroadcaster),
		admin.WithRealtimeEmitter(realtimePublisher),
		admin.WithCatalog(admin.NewCatalogAdapter(catalogStore)),
		admin.WithStorage(storageProvider, storageProvider),
		admin.WithRuntimeResolver(runtimeResolver),
		admin.WithRuntimeConfig(runtimeInvalidator{
			resolver:     runtimeResolver,
			r2:           r2Resolver,
			settingsRepo: settingsRepo,
		}),
		admin.WithPricingReload(func(ctx context.Context) {
			if rule, err := billingRepo.GetActiveRule(ctx); err == nil {
				pricingEngine.SetRule(rule)
			} else {
				logger.Warn("pricing reload after admin change failed", zap.Error(err))
			}
		}),
		admin.WithBreakerReload(func(ctx context.Context) {
			applyBreakerSettings(ctx, aiGateway, settingsRepo)
		}),
		admin.WithMeteringReload(func(ctx context.Context) {
			applyMeteringSettings(ctx, settingsRepo)
		}),
		// AI timeout budget: re-apply on the pod that served the change, then
		// fan the hint out so peers converge too (zero-downtime).
		admin.WithTimeoutsReload(func(ctx context.Context) {
			aiTuner.Apply(ctx)
		}),
		admin.WithSettingsBroadcaster(settingsPubSub),
	)
	superAdminH := admin.NewHandler(adminSvc)
	adminRoleResolver := admin.NewRoleResolver(pool, cacheEngine)

	r := deliveryHTTP.SetupRouter(
		routerConfig,
		transactionUC,
		categoryUC,
		userUC,
		dashboardUC,
		spkUC,
		spkMetricUC,
		budgetUC,
		paymentAccountUC,
		aiUC,
		providerUC,
		reportUC,
		authUC,
		entHandler,
		realtimeHandler,
		payHandler,
		superAdminH,
		adminRoleResolver,
	)

	// Start the SPK worker pool.
	spkQueue.Start(bgCtx)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
		// ReadTimeout bounds how long the client may take to send the request
		// (headers + body). It must stay comfortably above the largest upload we
		// accept: the receipt endpoint allows a 2 MB image, so 15s is too tight
		// on a slow mobile uplink.
		ReadTimeout: envSeconds("HTTP_READ_TIMEOUT", 60*time.Second),
		// WriteTimeout is the *whole-response* budget and is the trap that made
		// receipt analysis look like it "timed out at 30s": it silently aborted
		// any AI call that ran longer than this. It is raised to a generous
		// default here, and the long-lived AI/SSE routes additionally clear the
		// deadline per-request via middleware.NoWriteTimeout so they are never
		// cut off mid-answer.
		WriteTimeout: envSeconds("HTTP_WRITE_TIMEOUT", 120*time.Second),
		IdleTimeout:  envSeconds("HTTP_IDLE_TIMEOUT", 60*time.Second),
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

	// Drain window: long enough for in-flight requests (including SSE streams,
	// which the shutdown closes) to finish. Override with SHUTDOWN_TIMEOUT
	// (seconds) when a platform needs a different budget; keep it in sync with
	// the orchestrator's termination grace period.
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout())
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited gracefully")
}

// applyBreakerSettings reads the breaker tuning keys from settings and applies
// them to the running AI gateway. Called once at boot and again whenever the
// Super Admin changes them, so tuning takes effect without a restart. Missing
// keys keep the built-in defaults (the gateway already seeded them).
func applyBreakerSettings(ctx context.Context, gw *gateway.Service, settingsRepo *postgres.SettingsRepo) {
	if gw == nil || settingsRepo == nil {
		return
	}
	read := func(key string) (int64, bool) {
		v, err := settingsRepo.GetInt(ctx, key)
		if err != nil || v <= 0 {
			return 0, false
		}
		return v, true
	}

	if v, ok := read("breaker_failure_threshold"); ok {
		cfg := gw.BreakerConfig()
		cfg.FailureThreshold = int(v)
		gw.SetBreakerConfig(cfg)
	}
	if v, ok := read("breaker_cooldown_seconds"); ok {
		cfg := gw.BreakerConfig()
		cfg.Cooldown = time.Duration(v) * time.Second
		gw.SetBreakerConfig(cfg)
	}
	if v, ok := read("breaker_half_open_probes"); ok {
		cfg := gw.BreakerConfig()
		cfg.HalfOpenMaxProbes = int(v)
		gw.SetBreakerConfig(cfg)
	}

	routeThreshold, routeCooldown := gw.RouteBreakerConfig()
	changed := false
	if v, ok := read("route_breaker_threshold"); ok {
		routeThreshold = int(v)
		changed = true
	}
	if v, ok := read("route_breaker_cooldown_secs"); ok {
		routeCooldown = time.Duration(v) * time.Second
		changed = true
	}
	if changed {
		gw.SetRouteBreakerConfig(routeThreshold, routeCooldown)
	}
}

// applyMeteringSettings reads token metering heuristic parameters from settings
// and updates the process-wide estimator.
func applyMeteringSettings(ctx context.Context, settingsRepo *postgres.SettingsRepo) {
	if settingsRepo == nil {
		return
	}
	read := func(key string) (int, bool) {
		v, err := settingsRepo.GetInt(ctx, key)
		if err != nil || v <= 0 {
			return 0, false
		}
		return int(v), true
	}

	cfg := metering.CurrentConfig()
	changed := false
	if v, ok := read("metering_chars_per_token"); ok {
		cfg.CharsPerToken = v
		changed = true
	} else if v, ok := read("ai_estimate_chars_per_token"); ok {
		cfg.CharsPerToken = v
		changed = true
	}
	if v, err := settingsRepo.GetInt(ctx, "metering_safety_buffer_pct"); err == nil && v >= 0 {
		cfg.SafetyBufferPct = int(v)
		changed = true
	}
	if v, ok := read("metering_image_tokens"); ok {
		cfg.ImageTokens = v
		changed = true
	} else if v, ok := read("ai_estimate_vision_image_tokens"); ok {
		cfg.ImageTokens = v
		changed = true
	}
	if v, ok := read("metering_audio_tokens_per_sec"); ok {
		cfg.TokensPerSecond = v
		changed = true
	} else if v, ok := read("ai_estimate_voice_tokens_per_sec"); ok {
		cfg.TokensPerSecond = v
		changed = true
	}
	if v, ok := read("ai_estimate_voice_bytes_per_sec"); ok {
		cfg.AudioBytesPerSecond = v
		changed = true
	}
	if changed {
		metering.SetConfig(cfg)
	}
}

// runtimeInvalidator clears both the config resolver cache and the R2 store
// cache after a Super Admin settings change, so the new values apply immediately
// rather than after the cache TTL.
type runtimeInvalidator struct {
	resolver     *runtimeconfig.Resolver
	r2           *storage.DynamicR2Resolver
	settingsRepo *postgres.SettingsRepo
}

func (r runtimeInvalidator) Invalidate() {
	if r.resolver != nil {
		r.resolver.Invalidate()
	}
	if r.r2 != nil {
		r.r2.Invalidate()
	}
	if r.settingsRepo != nil {
		applyMeteringSettings(context.Background(), r.settingsRepo)
	}
}
