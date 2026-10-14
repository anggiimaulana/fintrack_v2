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
  },
  profile: {
    me: ["profile", "me"] as const,
  },
  entitlements: {
    me: ["entitlements", "me"] as const,
  },
  billing: {
    plans: ["billing", "plans"] as const,
    payments: ["billing", "payments"] as const,
    invoices: ["billing", "invoices"] as const,
    subscription: ["billing", "subscription"] as const,
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
    gatewayEndpoints: ["super-admin", "gateway-endpoints"] as const,
    gatewayRoutes: ["super-admin", "gateway-routes"] as const,
    killSwitch: ["super-admin", "kill-switch"] as const,
    auditLogs: (params?: unknown) => ["super-admin", "audit-logs", params] as const,
    settings: ["super-admin", "settings"] as const,
    featureFlags: ["super-admin", "feature-flags"] as const,
  },
};
