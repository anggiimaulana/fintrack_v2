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
  geminiKeys: {
    all: ["gemini-keys"] as const,
  },
  providers: {
    all: ["providers"] as const,
  },
};
