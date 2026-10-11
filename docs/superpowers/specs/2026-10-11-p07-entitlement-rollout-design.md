# P07 Slice A — Entitlement & RBAC Rollout: Design

**Date:** 2026-10-11
**Status:** Approved (pending written-spec review)
**Phase:** P07 (Rollout RBAC, Realtime & Standar) — first of five slices
**Related tasks:** TASK-P07-001, TASK-P07-002 (+ frontend mirror from `frontend-entitlement-mirror.md`)

---

## 1. Problem

The entitlement middleware already exists (`internal/delivery/http/middleware/entitlement.go`,
with tests), the resolver works (`internal/entitlement/`), and `GET /me/entitlements`
serves real data. But **`NewEntitlementMiddleware` is never constructed** — `main.go:225`
builds the resolver and the handler, and the router wires only `AuthMiddleware` +
`RequireUser`. Result: **0 of 63 routes are entitlement-gated.**

This is the worst possible state: the capability looks implemented (unit tests pass) while
enforcing nothing. Every legacy endpoint is protected by *authentication* only, so any
signed-in user can reach pro/unlimited features.

Two further defects surfaced during reconnaissance:

### 1.1 Enforcing now would lock out ~96% of users

`entitlement/resolver.go:105` returns **deny-all** when a user has no subscription:

```go
if plan == nil {
    // No plan -> deny-all (fail-closed).
    return res, nil
}
```

`EntitlementSource.GetActivePlanForUser` returns `nil` on `pgx.ErrNoRows`, and
`AuthUsecase.Register` creates a user with **no subscription row**. Dev database:
**52 users, 2 subscriptions** — 50 users would receive 403 on every AI feature and 402 on
every limit the moment enforcement is switched on. There is no fallback to the Free plan.

### 1.2 The P07 documentation does not match the codebase

`teknis/rbac-rollout.md` documents a 28-endpoint matrix containing endpoints that do not
exist (`/budgets`, `/goals`, `/analytics/cashflow`, `/analytics/predict`,
`/transactions/export`, `/ai/receipt-scan`, `/ai/token-balance`, `/user/profile`). It also
specifies **429** for limit exhaustion, while the code and its tests use **402**
(`TestWithinLimit_Exceeded402`, plus an explicit fail-closed test for absent keys).

**Decision (user-approved):** protect the routes that actually exist; keep 402 and correct
the documentation.

---

## 2. Goals / Non-goals

**Goals**

1. Wire entitlement enforcement onto the existing user-facing routes that correspond to
   seeded feature/limit keys.
2. Make users without a subscription resolve to the **Free** plan so enforcement cannot lock
   them out.
3. Provide a frontend mirror (`useEntitlements`, `useRole`) and a read-only subscription
   page so `upgrade_hint` links somewhere real.
4. Correct the P07 documentation to describe the code that exists.

**Non-goals**

- No new product endpoints (budgets, goals, analytics, export). P07 is a rollout phase.
- No user-facing billing/purchase flow.
- No removal of BYOK (`/gemini-keys*`) — that is slice D.
- No SSE/realtime work — slice B.

---

## 3. Design

### 3.1 Free-plan fallback (unblocks enforcement)

**Change** in `internal/repository/postgres/entitlement_source.go`
(`GetActivePlanForUser`): when no `subscriptions` row exists, fall back to the plan whose
`code` equals `DEFAULT_PLAN_CODE` (default `free`). The returned status is the explicit
sentinel `billing.StatusNone` — a new constant with value `"none"`, distinct from every
lifecycle state (`trialing`, `active`, `past_due`, `grace`, `canceled`, `expired`). This lets
the UI distinguish "never subscribed, on Free" from "paid plan that lapsed" without
inferring it from an unrelated lifecycle value. `Resolved.Status` has no existing consumer,
so adding the sentinel is additive.

**Rationale.** The seed data defines `free` as the entry tier, and Free's entitlements are
already deliberately restrictive (`ai_chat: true`, `ocr_receipt: false`, `voice_input: false`,
`reports_export: false`, `max_transactions: 100`). That *is* the product's default tier; the
defect is that new users were never assigned to it. Deny-all is not fail-closed for a
*missing default* — it is a data-initialization gap.

**Fail-closed is preserved.** If the `free` plan itself is missing or unreadable, resolution
continues to return deny-all and logs the condition. Access is never widened on error.

**Blast radius.** `entitlement.Source` has exactly two implementers:
`postgres.EntitlementSource` and the test fakes in `resolver_test.go` /
`entitlement_test.go`.

**No migration.** Because the resolver falls back to Free, the 50 subscription-less users are
covered without a backfill, and new registrations need no subscription row.

### 3.2 Enforcement matrix

Gating is applied per-route with `.Use(...)`, not to the shared `userRoutes` group, so read
routes keep their existing 30-second cache behaviour untouched.

| Route | Gate | Key | Free value |
|---|---|---|---|
| `POST /transactions` | RequireLimit | `max_transactions` | 100 |
| `POST /ai/chat` | RequireFeature + RequireLimit | `ai_chat` / `ai_tokens_per_cycle` | true / 5000 |
| `POST /ai/receipt` | RequireFeature + RequireLimit | `ocr_receipt` / `ocr_scans_per_cycle` | false / 0 |
| `POST /ai/receipt/transaction` | RequireFeature | `ocr_receipt` | false |
| `POST /ai/voice` | RequireFeature | `voice_input` | false |
| `GET /reports` | RequireFeature | `reports_export` | false |

`max_transactions` is interpreted as the user's **total transaction count** (the literal
reading of the key). Usage resolvers:

| Limit key | Usage source |
|---|---|
| `max_transactions` | `COUNT(*)` of the user's transactions where `is_deleted = false` |
| `ai_tokens_per_cycle` | current period's `used_tokens + reserved_tokens` |
| `ocr_scans_per_cycle` | `token_ledger` rows with `modality='vision'` in the current period |

**"Current period"** means the row where `period_start <= now <= period_end`, ordered by
`period_start DESC` — the same predicate the ledger already uses. The resolver needs a
**non-locking read**: the existing `GetPeriodForUpdate` takes a row lock, which is
inappropriate for a pre-flight check, so a read-only `GetCurrentPeriod(ctx, userID, now)` is
added to the `TokenRepository` interface, its Postgres implementation, and the test fakes.

**Absent data vs. error — deliberately different:**

| Situation | Behaviour |
|---|---|
| No current token period exists | usage = **0**, request **permitted**. The ledger creates the period on first use; denying here would lock out every user who has not yet made an AI call. |
| No vision ledger rows in period | usage = 0, permitted |
| Query error / database unreachable | **500**, request denied (never widen access on error) |

This distinction matters: a blanket "fail-closed when data is unavailable" would reproduce
the same lockout class of bug that §3.1 fixes.

**Deliberately left open:** every read route (`GET /transactions`, `/dashboard/*`,
`/transactions/stats`, `/categories/*`, `/spk/*`, `/ai/sessions*`, `/providers`), plus
`POST /ai/sessions` and session update/delete (session bookkeeping is not a metered feature).
`POST /spk/calculate` is not entitlement-gated: no seeded key matches it and it is already
rate-limited at 30/min.

### 3.3 Seeded keys intentionally NOT enforced in this slice

Recorded explicitly so they are not mistaken for complete:

| Key | Reason |
|---|---|
| `ai_requests_per_cycle` | No request counter exists. `token_periods` has only `used_tokens`/`reserved_tokens`; ledger `entry_type` values are `reserve/commit/release/adjust` — no per-request rows. Needs new counting machinery. |
| `max_categories` | Categories are super-admin-managed; there are no user-facing category write routes. |
| `max_wallets` | No wallets feature or routes exist. |
| `rag_search` | No RAG endpoint; RAG is embedded in `/ai/chat`, already gated by `ai_chat`. |
| `advanced_budgeting`, `api_access`, `bank_sync` | No corresponding routes/features yet. |

### 3.4 Block contract

| Condition | Status | Error code |
|---|---|---|
| Feature not in plan | 403 | `feature_not_in_plan` |
| Limit exhausted | 402 | `limit_exceeded` |
| Limit key absent from plan | 402 | `limit_not_in_plan` (fail-closed) |

402 is retained because it is already implemented and tested, and because the rate limiter
already returns 429 — using 429 for quota exhaustion would make "going too fast" and "out of
quota" indistinguishable to clients. The documentation is corrected to match.

Both 403 and 402 payloads gain `upgrade_url: "/user/settings/subscription"` (currently only
`upgrade_hint: true` is returned).

### 3.5 Rollout valve

`ENTITLEMENT_ENFORCEMENT` env var — default **on**; when `off`, the middleware no-ops and
logs. Included because P07's README calls for a gradual-rollout / instant-kill-switch
capability. Documented explicitly as an operational bypass: an operator with env access can
disable authorization, so it must not be set to `off` in production except during an incident.

### 3.6 Frontend mirror

- `frontend/hooks/use-entitlements.ts` — one TanStack Query against the existing
  `GET /me/entitlements`, reusing the shared axios client (httpOnly cookies + single-flight
  401 refresh already handled). Exposes `hasFeature(key)` and `withinLimit(key, current?)`.
  Query key registered in the existing `lib/cache-keys.ts`.
- `frontend/hooks/use-role.ts` — typed wrapper over the existing `useSession()`, returning
  `normalizeRole(user?.role)` from the existing `lib/roles.ts`. Does **not** re-fetch `/me`.
- UX mirror is **advisory only**: gated menu items/actions render disabled with an upgrade
  hint; 403/402 responses surface as toasts linking to the subscription page. `proxy.ts`
  continues to guard by role, and the backend remains the sole entitlement authority.
- New page `/user/settings/subscription` — read-only, driven by `useEntitlements()`:
  plan name + status, feature list with enabled/disabled state, limits table (`-1` rendered
  as "Unlimited"). No purchase flow.

---

## 4. Testing

### 4.1 Backend unit

`internal/entitlement`: no-subscription → Free features/limits (not empty); active Pro
unchanged; `free` plan missing → deny-all; cache/version behaviour unchanged.

`internal/delivery/http/middleware`: 403 feature-denied; 402 limit-exceeded; 402 fail-closed
on absent key; super_admin bypass; usage-resolver error → 500 (never allow); **no current
token period → usage 0 and request permitted** (not 402).

### 4.2 Backend integration (DB-backed, `TEST_DATABASE_URL`)

Boots the **real router** and asserts enforcement end-to-end — the regression guard for the
original defect, which unit tests could not catch because they exercised the middleware
directly rather than its wiring:

- Free user → `GET /reports` = 403 `feature_not_in_plan`
- Pro user → `GET /reports` = 200
- Free user at 100 transactions → `POST /transactions` = 402 `limit_exceeded`
- Free user at 99 transactions + 1 soft-deleted → still permitted (soft-deleted rows excluded)
- Free user with **no token period** → `POST /ai/chat` permitted (usage 0, not 402)
- Free user → `POST /ai/voice` = 403; Pro user → 200
- super_admin → bypasses all gates
- User with **no subscription row** → treated as Free (allowed `ai_chat`, blocked `ocr_receipt`)
- 403/402 payloads include `upgrade_url`

### 4.3 Frontend

Vitest for `use-entitlements` (feature/limit logic, `-1` = unlimited, fail-closed on null) and
`use-role`; a render test for the subscription page. `pnpm lint`, `pnpm build`, `pnpm test` green.

---

## 5. Documentation corrections

- `P07/teknis/rbac-rollout.md` — replace the aspirational 28-endpoint matrix with the real
  routes; correct 429 → 402; use real key names (`max_transactions`, not
  `monthly_transactions`); document `upgrade_url`; state which keys remain unenforced.
- `P07/planning/tasks.md` — TASK-P07-001 acceptance criteria (429 → 402).
- `P07/tracking/tracking.md` — status for TASK-P07-001/002 and the frontend mirror.
- `second_brain/1_Digest/Tugas-Rutin/log_aktivitas.md` — activity entry.

---

## 6. Risks

| Risk | Mitigation |
|---|---|
| Enforcement locks out existing users | Free-plan fallback (§3.1) + integration test for the no-subscription case + `ENTITLEMENT_ENFORCEMENT` valve (§3.5) |
| Existing users already above the Free limit are blocked from *creating* (e.g. demo users hold 479 transactions vs. Free's 100). Reads stay open; only `POST /transactions` is refused | Accepted and intentional — it is correct enforcement. Call out in the activity log; the valve (§3.5) is the rollback if it proves disruptive |
| `free` plan deleted/renamed | Fail-closed deny-all + log; `DEFAULT_PLAN_CODE` configurable |
| Middleware silently un-wired again | Integration test asserts a Free user is blocked on a gated route |
| Frontend treated as authority | Mirror is advisory; backend remains sole enforcer; `proxy.ts` guards by role |
| Enforcement bypassed via env | Documented as incident-only; not a runtime toggle exposed to users |
