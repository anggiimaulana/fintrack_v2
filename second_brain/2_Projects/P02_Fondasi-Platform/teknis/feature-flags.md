# Dokumen Teknis — Dynamic Feature Flags

## 1. Ikhtisar Arsitektur
Feature Flags di FinTrack v2 memungkinkan aktivasi/deaktivasi fitur baru secara selektif, uji coba canary rollout bertahap, dan pengaktifan mekanisme darurat (*emergency kill-switch*) tanpa perlu melakukan deploy ulang kode binary backend maupun frontend.

Arsitektur ini mendukung prinsip **Zero Downtime (K-23)** dan **Fail-Closed Security (K-P02-07)**.

---

## 2. Struktur Database: Tabel `feature_flags`

### 2.1. File Migrasi `0006_create_feature_flags.sql`
```sql
-- Up Migration: 0006_create_feature_flags.sql

CREATE TABLE IF NOT EXISTS feature_flags (
    id SERIAL PRIMARY KEY,
    key VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    strategy VARCHAR(32) NOT NULL DEFAULT 'BOOLEAN', 
    percentage INT NOT NULL DEFAULT 0,
    rules JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_feature_flags_key ON feature_flags(key);

-- Trigger auto-update timestamp
CREATE OR REPLACE FUNCTION update_feature_flags_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_feature_flags_updated_at
BEFORE UPDATE ON feature_flags
FOR EACH ROW
EXECUTE FUNCTION update_feature_flags_timestamp();
```

---

## 3. Strategi Evaluasi Feature Flag

Modul `internal/featureflag` mendukung 4 strategi penentuan:

| Strategi | Parameter Terkait | Mekanisme Evaluasi | Use Case |
|---|---|---|---|
| `BOOLEAN` | `is_enabled` | True/False biner global. | Global Kill Switch, fitur umum. |
| `PERCENTAGE` | `percentage` (0–100) | Hashing deterministik: `crc32(key + user_id) % 100 < percentage`. | Canary Rollout rilis AI Gateway baru. |
| `USER_LIST` | `rules.allowed_users` | Pencocokan string UUID user dalam array JSON. | Uji coba internal tim / beta tester khusus. |
| `ROLE` | `rules.roles` | Pengecekan role pengguna (`super_admin` vs `user`). | Fitur awal untuk admin sebelum publik. |

---

## 4. Evaluasi Runtime & Caching Redis L2

### 4.1. Alur Evaluasi
1. Evaluator memeriksa status flag di Redis hash `ff:flags` (key: flag key, val: serialized JSON).
2. Jika key tidak ditemukan di Redis (cache miss):
   - Ambil dari PostgreSQL tabel `feature_flags`.
   - Simpan ke Redis dengan TTL 10 menit.
3. Evaluator menjalankan logika strategi berdasarkan konteks pemanggil (`userID`, `userRole`).
4. **Prinsip Fail-Closed**: Jika Redis dan DB mengalami gangguan bersamaan, evaluator mengembalikan `false` secara default (fitur dinonaktifkan demi keselamatan data).

### 4.2. Implementasi Go Engine Evaluator
```go
package featureflag

import (
	"context"
	"database/sql"
	"encoding/json"
	"hash/crc32"
	"time"

	"github.com/redis/go-redis/v9"
)

type Flag struct {
	Key        string          `json:"key"`
	IsEnabled  bool            `json:"is_enabled"`
	Strategy   string          `json:"strategy"`
	Percentage int             `json:"percentage"`
	Rules      json.RawMessage `json:"rules"`
}

type Context struct {
	UserID   string
	UserRole string
}

type Evaluator struct {
	db  *sql.DB
	rdb *redis.Client
}

func NewEvaluator(db *sql.DB, rdb *redis.Client) *Evaluator {
	return &Evaluator{db: db, rdb: rdb}
}

func (e *Evaluator) IsEnabled(ctx context.Context, flagKey string, evalCtx Context) bool {
	flag, err := e.getFlag(ctx, flagKey)
	if err != nil || !flag.IsEnabled {
		return false // Fail-closed
	}

	switch flag.Strategy {
	case "BOOLEAN":
		return flag.IsEnabled

	case "PERCENTAGE":
		if evalCtx.UserID == "" {
			return false
		}
		hashVal := crc32.ChecksumIEEE([]byte(flagKey + evalCtx.UserID))
		bucket := int(hashVal % 100)
		return bucket < flag.Percentage

	case "ROLE":
		var ruleData struct {
			Roles []string `json:"roles"`
		}
		if err := json.Unmarshal(flag.Rules, &ruleData); err != nil {
			return false
		}
		for _, r := range ruleData.Roles {
			if r == evalCtx.UserRole {
				return true
			}
		}
		return false

	default:
		return false
	}
}

func (e *Evaluator) getFlag(ctx context.Context, key string) (*Flag, error) {
	// Coba dari Redis Cache
	cached, err := e.rdb.Get(ctx, "ff:"+key).Result()
	if err == nil {
		var flag Flag
		if json.Unmarshal([]byte(cached), &flag) == nil {
			return &flag, nil
		}
	}

	// Fallback ke PostgreSQL
	var flag Flag
	query := `SELECT key, is_enabled, strategy, percentage, rules FROM feature_flags WHERE key = $1`
	row := e.db.QueryRowContext(ctx, query, key)
	if err := row.Scan(&flag.Key, &flag.IsEnabled, &flag.Strategy, &flag.Percentage, &flag.Rules); err != nil {
		return nil, err
	}

	// Simpan ke Redis (TTL 10 menit)
	bytes, _ := json.Marshal(flag)
	_ = e.rdb.Set(ctx, "ff:"+key, bytes, 10*time.Minute).Err()

	return &flag, nil
}
```

---

## 5. Invalidation & Runtime Update
Ketika Super Admin memperbarui nilai flag di dashboard:
1. Row diperbarui di database PostgreSQL.
2. Backend menghapus key Redis `ff:{key}` dan mempublikasikan notifikasi ke channel Pub/Sub `cache:invalidate`.
3. Seluruh instance aplikasi langsung mengevaluasi konfigurasi baru pada request milidetik berikutnya.
