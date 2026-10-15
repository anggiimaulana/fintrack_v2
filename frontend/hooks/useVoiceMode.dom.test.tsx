import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import * as React from "react";

// ─── module mocks ──────────────────────────────────────────────────

const entitlementsState: { data: unknown } = { data: null };
vi.mock("@/hooks/use-entitlements", () => ({
  useEntitlements: () => ({ entitlements: entitlementsState.data }),
}));

const configState: { data: unknown } = { data: null };
vi.mock("@/hooks/useVoiceConfig", () => ({
  useVoiceConfig: () => ({ config: configState.data }),
}));

import { useVoiceMode } from "./useVoiceMode";

function wrapper(client: QueryClient) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

function newClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

/** Minimal config shape the hook reads. */
function cfg(over: Record<string, unknown> = {}) {
  return {
    max_duration_seconds: 60,
    language_mode: "id_only",
    default_language: "id",
    languages: ["id"],
    language_per_recording: false,
    daily_limit: { used: 0, limit: 50, remaining: 50, unlimited: false },
    daily_reset_timezone: "Asia/Jakarta",
    daily_resets_at: "",
    history_retention_days: 90,
    allow_edit_before_send: true,
    vu_meter_enabled: true,
    fallback_to_browser: true,
    server_enabled: true,
    server_available: true,
    ...over,
  };
}

function ent(over: Record<string, unknown> = {}) {
  return {
    features: { voice_input: true },
    limits: {},
    usage: {
      ai_tokens_per_cycle: { used: 0, limit: 1000, remaining: 1000, unlimited: false },
      ai_requests_per_cycle: { used: 0, limit: 100, remaining: 100, unlimited: false },
    },
    ...over,
  };
}

beforeEach(() => {
  entitlementsState.data = null;
  configState.data = null;
});

describe("useVoiceMode", () => {
  it("does NOT flash daily_limit while config is still loading", async () => {
    // Both sources are null (still loading). The mic must not appear blocked.
    const { result } = renderHook(() => useVoiceMode(), { wrapper: wrapper(newClient()) });
    await waitFor(() => expect(result.current).toBeTruthy());
    expect(result.current.mode).not.toBe("daily_limit");
  });

  it("uses server mode when the plan and quota allow it", async () => {
    entitlementsState.data = ent();
    configState.data = cfg();
    const { result } = renderHook(() => useVoiceMode(), { wrapper: wrapper(newClient()) });
    // Capabilities are detected in an effect; wait for the recorder to appear.
    await waitFor(() => expect(result.current.capabilities.recorder).toBeDefined());
  });

  it("blocks with daily_limit when the daily cap is exhausted", async () => {
    entitlementsState.data = ent();
    configState.data = cfg({ daily_limit: { used: 50, limit: 50, remaining: 0, unlimited: false } });
    const { result } = renderHook(() => useVoiceMode(), { wrapper: wrapper(newClient()) });
    await waitFor(() => expect(result.current.mode).toBe("daily_limit"));
    expect(result.current.canRecord).toBe(false);
  });

  it("treats a zero limit (key not in plan) as no headroom", async () => {
    entitlementsState.data = ent();
    configState.data = cfg({ daily_limit: { used: 0, limit: 0, remaining: 0, unlimited: false } });
    const { result } = renderHook(() => useVoiceMode(), { wrapper: wrapper(newClient()) });
    await waitFor(() => expect(result.current.mode).toBe("daily_limit"));
  });

  it("treats unlimited as headroom", async () => {
    entitlementsState.data = ent();
    configState.data = cfg({ daily_limit: { used: 999, limit: -1, remaining: -1, unlimited: true } });
    const { result } = renderHook(() => useVoiceMode(), { wrapper: wrapper(newClient()) });
    await waitFor(() => expect(result.current.mode).not.toBe("daily_limit"));
    expect(result.current.dailyRemaining).toBe(-1);
  });

  it("falls back to browser when the server kill switch is off", async () => {
    entitlementsState.data = ent();
    configState.data = cfg({ server_enabled: false });
    const { result } = renderHook(() => useVoiceMode(), { wrapper: wrapper(newClient()) });
    await waitFor(() => expect(result.current.isPremium).toBe(false));
  });
});
