export const queryKeys = {
  transactions: {
    all: ["transactions"] as const,
    list: (filter?: string) => ["transactions", "list", filter] as const,
    stats: (period?: string) => ["transactions", "stats", period] as const,
    detail: (id: string) => ["transactions", "detail", id] as const,
  },
  dashboard: {
    all: ["dashboard"] as const,
    overview: (period?: string) => ["dashboard", "overview", period] as const,
    charts: (period?: string) => ["dashboard", "charts", period] as const,
  },
  spk: {
    all: ["spk"] as const,
    latest: ["spk", "latest"] as const,
    history: ["spk", "history"] as const,
  },
  categories: {
    all: ["categories"] as const,
    income: ["categories", "income"] as const,
    expense: ["categories", "expense"] as const,
    savings: ["categories", "savings"] as const,
    subcategories: (id: string) => ["categories", "subcategories", id] as const,
  },
  ai: {
    history: ["ai", "history"] as const,
    // Effective AI timeout budget, resolved server-side from the dynamic
    // settings so the client never hardcodes a value.
    config: ["ai", "config"] as const,
  },
  admin: {
    users: ["admin", "users"] as const,
    stats: ["admin", "stats"] as const,
    recentUsers: ["admin", "recent-users"] as const,
    categories: (type: string) => ["admin", "categories", type] as const,
    subcategories: ["admin", "subcategories"] as const,
  },
  reports: {
    all: ["reports"] as const,
    byPeriod: (period: string) => ["reports", period] as const,
    analytics: (params: string) => ["reports", "analytics", params] as const,
  },
  spkMetrics: {
    all: ["spk-metrics"] as const,
    mine: ["spk-metrics", "mine"] as const,
    default: ["spk-metrics", "default"] as const,
  },
  budgets: {
    all: ["budgets"] as const,
  },
  paymentAccounts: {
    all: ["payment-accounts"] as const,
    active: ["payment-accounts", "active"] as const,
  },
  profile: {
    me: ["profile", "me"] as const,
  },
  notifications: {
    all: ["notifications"] as const,
    list: (params?: string) => ["notifications", "list", params] as const,
    stats: ["notifications", "stats"] as const,
    detail: (id: string) => ["notifications", "detail", id] as const,
    preferences: ["notifications", "preferences"] as const,
  },
  entitlements: {
    me: ["entitlements", "me"] as const,
  },
  voice: {
    config: ["voice", "config"] as const,
    /** Prefix key: invalidating this matches every paged history query. */
    historyAll: ["voice", "history"] as const,
    history: (params?: string) => ["voice", "history", params] as const,
  },
  usage: {
    all: ["usage"] as const,
    timeseries: (range: string) => ["usage", "timeseries", range] as const,
    history: (params: string) => ["usage", "history", params] as const,
  },
  billing: {
    plans: ["billing", "plans"] as const,
    payments: ["billing", "payments"] as const,
    invoices: ["billing", "invoices"] as const,
    subscription: ["billing", "subscription"] as const,
    paymentsPaged: (params: string) => ["billing", "payments", "paged", params] as const,
    invoicesPaged: (params: string) => ["billing", "invoices", "paged", params] as const,
    subscriptionEvents: (params: string) => ["billing", "subscription", "events", params] as const,
  },
  providers: {
    all: ["providers"] as const,
  },
  superAdmin: {
    metricsOverview: (refresh?: boolean) => ["super-admin", "metrics-overview", refresh] as const,
    metricsTimeseries: (period?: string) => ["super-admin", "metrics-timeseries", period] as const,
    metricsPaymentMethods: ["super-admin", "metrics-payment-methods"] as const,
    plans: ["super-admin", "plans"] as const,
    subscriptions: (params?: unknown) => ["super-admin", "subscriptions", params] as const,
    payments: (params?: unknown) => ["super-admin", "payments", params] as const,
    paymentsReview: (params?: unknown) => ["super-admin", "payments-review", params] as const,
    tokensUsers: (params?: unknown) => ["super-admin", "tokens-users", params] as const,
    tokensLedger: (params?: unknown) => ["super-admin", "tokens-ledger", params] as const,
    // Must match gatewayKeys in hooks/useSuperAdminGateway.ts exactly, otherwise
    // a realtime invalidation targets a key no query uses (silent no-op).
    gatewayEndpoints: ["super-admin", "ai-gateway", "endpoints"] as const,
    gatewayRoutes: ["super-admin", "ai-gateway", "routes"] as const,
    killSwitch: ["super-admin", "ai-gateway", "kill-switch"] as const,
    auditLogs: (params?: unknown) => ["super-admin", "audit-logs", params] as const,
    settings: ["super-admin", "settings"] as const,
    featureFlags: ["super-admin", "feature-flags"] as const,
    // Dynamic configuration catalog (P-13). Keys MUST match configCatalogKeys
    // in hooks/useConfigCatalog.ts so a config_changed invalidation lands.
    entitlementCatalog: ["admin", "entitlement-catalog"] as const,
    settingDefinitions: ["admin", "setting-definitions"] as const,
    metricDefinitions: ["admin", "metric-definitions"] as const,
    featureDefinitions: ["admin", "feature-definitions"] as const,
    resolvers: ["admin", "resolvers"] as const,
  },
};
