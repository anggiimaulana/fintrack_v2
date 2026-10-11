# Autentikasi & Otorisasi — FinTrack v2

## 1. Model Identitas

| Aspek | Nilai |
|---|---|
| Role | `super_admin`, `user` |
| Sumber role | `users.role` |
| Admin lama | Dipromosikan menjadi `super_admin` (migrasi data) |
| Multi-role | Tidak (satu role per user) |

## 2. Autentikasi (Go-native)

```
Login → verifikasi bcrypt(cost 12) → terbitkan:
  - Access token  (JWT HS256, 15 menit, cookie ft_at, httpOnly)
  - Refresh token (opaque, hash sha256 di DB, cookie ft_rt, rotasi)
```

- Semua trafik browser lewat **proxy same-origin** Next.js.
- Refresh token: rotasi + deteksi reuse (`replaced_by`).
- Rate limit login (default 5/menit, overridable via env).

## 3. Otorisasi Dua Lapis

### 3.1 Lapis role (identitas)

- `RequireAdmin()`/`RequireUser()` dipertahankan, `RequireSuperAdmin()` ditambahkan.
- Hanya mengecek **siapa** kamu.

### 3.2 Lapis entitlement (hak akses)

- Diturunkan dari `plan_entitlements` (fitur bool) + `plan_limits` (numerik) + `subscriptions`.
- **Dipisah dari role** — prinsip RBAC ≠ billing.
- Middleware: `RequireFeature(featureKey)`, `WithinLimit(limitKey)`.

## 4. Resolusi Entitlement (realtime, tanpa login/logout)

```
Request → AuthMiddleware (identitas saja)
        → EntitlementGuard.Require(feature|limit)
              └─ baca cache key "ent:<userID>:v<version>"
                 miss → hitung dari DB → cache TTL pendek
```

- **JWT tidak memuat entitlement** (keputusan K-14).
- Super admin ubah centang → **bump entitlement_version** → cache lama gugur → request berikutnya
  memakai aturan baru. **Tanpa login/logout, tanpa restart.**

## 5. Matriks Akses (contoh)

| Aksi | super_admin | user |
|---|:--:|:--:|
| Kelola plan/entitlement | ✅ | ❌ |
| Kelola endpoint AI & kunci | ✅ | ❌ |
| Review pembayaran manual | ✅ | ❌ |
| Lihat audit log | ✅ | ❌ |
| Catat transaksi | ✅ | ✅ |
| Pakai AI (sesuai plan) | ✅ | ✅ (entitlement) |
| Kelola langganan sendiri | ✅ | ✅ |

## 6. Aturan

1. Backend otoritatif; frontend hanya menyembunyikan/menonaktifkan UI.
2. Setiap endpoint sensitif wajib melewati guard yang sesuai.
3. Perubahan entitlement tidak boleh memerlukan migrasi data user.
4. Semua aksi super_admin dicatat ke audit log.

## Referensi

- `../../2_Projects/P04_Subscription-Token-Core/teknis/enforcement-middleware.md`
- `../../2_Projects/P07_Rollout-RBAC-Realtime/teknis/rbac-rollout.md`
