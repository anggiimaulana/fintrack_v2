# Diagram — ERD FinTrack v2 (Setelah Penambahan)

> Tabel lama dipertahankan. Yang baru ditambahkan untuk subscription, token, pembayaran, AI gateway,
> dan platform. `gemini_api_keys` diarsipkan (tetap ada), `transaction_embeddings` tetap `vector(768)`.

```mermaid
erDiagram
    users ||--o{ subscriptions : memiliki
    users ||--o{ payments : melakukan
    users ||--o{ invoices : menerima
    users ||--o{ token_ledger : mencatat
    users ||--o{ token_periods : punya
    users ||--o{ transactions : mencatat
    users ||--o{ refresh_tokens : punya
    users ||--o{ chat_sessions : punya
    users ||--o{ gemini_api_keys : "arsip"

    plans ||--o{ plan_entitlements : punya
    plans ||--o{ plan_limits : punya
    plans ||--o{ subscriptions : dipilih
    plans ||--o{ pricing_rules : "override"

    subscriptions ||--o{ subscription_events : mencatat
    subscriptions ||--o{ payments : dibayar
    subscriptions ||--o{ invoices : ditagih
    subscriptions ||--o{ token_periods : siklus

    token_periods ||--o{ token_ledger : entri

    ai_endpoints ||--o{ ai_routes : dirutekan

    users {
        uuid id PK
        text email
        text role "super_admin|user"
        bool is_active
    }
    plans {
        uuid id PK
        text code
        text name
        int price_idr
        int billing_period_days
        bool is_active
        text gateway_override
    }
    plan_entitlements {
        uuid id PK
        uuid plan_id FK
        text feature_key
        bool enabled
    }
    plan_limits {
        uuid id PK
        uuid plan_id FK
        text limit_key
        int value
    }
    subscriptions {
        uuid id PK
        uuid user_id FK
        uuid plan_id FK
        text status
        timestamptz current_period_start
        timestamptz current_period_end
        text gateway
        bool cancel_at_period_end
    }
    subscription_events {
        uuid id PK
        uuid subscription_id FK
        text event_type
        jsonb before
        jsonb after
        timestamptz created_at
    }
    token_ledger {
        uuid id PK
        uuid user_id FK
        uuid period_id FK
        text entry_type "reserve|commit|release|grant|adjust"
        int amount
        text modality
        text ref_type
        uuid ref_id
        text idempotency_key UK
        timestamptz created_at
    }
    token_periods {
        uuid id PK
        uuid user_id FK
        timestamptz period_start
        timestamptz period_end
        int quota
        int used
        int reserved
    }
    pricing_rules {
        uuid id PK
        text scope "global|plan"
        uuid plan_id FK
        int base_per_1k_idr
        numeric markup_multiplier
        numeric markup_percent
        jsonb modality_multipliers
    }
    payments {
        uuid id PK
        uuid user_id FK
        uuid subscription_id FK
        text gateway "midtrans|manual"
        text gateway_ref
        int amount_idr
        text status
        text method
        text proof_url
        text idempotency_key UK
    }
    invoices {
        uuid id PK
        text number UK
        uuid user_id FK
        uuid subscription_id FK
        int subtotal_idr
        int tax_idr
        int total_idr
        text status
        timestamptz issued_at
        timestamptz due_at
        timestamptz paid_at
    }
    ai_endpoints {
        uuid id PK
        text base_url
        text api_key_encrypted
        text provider
        int priority
        text status
        text health
        timestamptz last_check_at
    }
    ai_routes {
        uuid id PK
        text use_case "chat|vision|voice|embedding|reasoning"
        uuid endpoint_id FK
        text model
        text combo
        int priority
        bool is_active
    }
    feature_flags {
        uuid id PK
        text key UK
        bool enabled
        int rollout
        uuid updated_by
    }
    audit_logs {
        uuid id PK
        uuid actor_id
        text action
        text entity
        uuid entity_id
        jsonb before
        jsonb after
        text ip
        text ua
        timestamptz created_at
    }
    settings {
        uuid id PK
        text key UK
        jsonb value
        uuid updated_by
        timestamptz updated_at
    }
```

**Catatan integritas:**
- Semua FK uang menuju `users`/`subscriptions`; `ON DELETE` dipilih hati-hati (data billing tidak
  ikut terhapus sembarangan).
- `token_ledger` **append-only**: tidak ada `UPDATE`/`DELETE` (kecuali kebijakan retensi tertulis).
- `payments.idempotency_key` dan `token_ledger.idempotency_key` **UNIQUE** → anti double-charge.
- Nilai uang = **integer** (`price_idr`, `amount_idr`, `total_idr`).
