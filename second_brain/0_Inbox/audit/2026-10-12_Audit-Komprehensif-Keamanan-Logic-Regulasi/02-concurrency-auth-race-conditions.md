# Temuan Konkurensi & Keamanan: Race Condition pada Rotasi Refresh Token

**Kode Dokumen**: AUDIT-SEC-03  
**Klasifikasi**: Autentikasi, Manajemen Sesi & Race Condition (Concurrency & Session Security)  

---

## 1. Lokasi Sumber Masalah
- **Berkas 1**: `backend/internal/repository/postgres/refresh_token_repo.go` (Baris 53–84)
- **Berkas 2**: `backend/internal/usecase/auth_usecase.go` (Baris 132–175)

```go
// backend/internal/repository/postgres/refresh_token_repo.go:62-67
func (r *RefreshTokenRepo) Rotate(ctx context.Context, oldID uuid.UUID, newToken *domain.RefreshToken) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var revokedAt *time.Time
	if err := tx.QueryRow(ctx,
		`SELECT revoked_at FROM refresh_tokens WHERE id = $1 FOR UPDATE`, oldID,
	).Scan(&revokedAt); err != nil {
		return fmt.Errorf("failed to lock refresh token: %w", err)
	}

	// [CELAH DI SINI]: Variabel revokedAt di-scan dari database di bawah FOR UPDATE lock,
	// TETAPI nilainya TIDAK PERNAH DIPERIKSA (revokedAt != nil)!
	// Eksekusi langsung melanjutkan ke INSERT dan UPDATE di bawah ini!

	if err := tx.QueryRow(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at, user_agent)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		newToken.UserID, newToken.TokenHash, newToken.ExpiresAt, newToken.UserAgent,
	).Scan(&newToken.ID, &newToken.CreatedAt); err != nil {
		return fmt.Errorf("failed to insert rotated token: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now(), replaced_by = $2, last_used_at = now() WHERE id = $1`,
		oldID, newToken.ID); err != nil {
		return fmt.Errorf("failed to revoke old token: %w", err)
	}

	return tx.Commit(ctx)
}
```

---

## 2. Analisis Mekanisme Celah & Race Condition

### 2.1 Alur Rotasi Token yang Ada Saat Ini
Di `AuthUsecase.Refresh`:
1. `existing, err := u.refreshRepo.GetByHash(ctx, hash)` dilakukan **di luar transaksi** (tanpa row lock).
2. Usecase memeriksa: `if existing.RevokedAt != nil { u.refreshRepo.RevokeAllForUser(ctx, existing.UserID); return ErrInvalidRefresh }`.
3. Jika lolos, usecase memanggil `u.refreshRepo.Rotate(ctx, existing.ID, newToken)`.
4. Di dalam `Rotate`, transaksi database dibuka dan mengambil row lock `SELECT revoked_at FROM refresh_tokens WHERE id = $1 FOR UPDATE`.
5. **Namun**, nilai `revokedAt` yang dibaca di bawah lock **diabaikan dan tidak divalidasi**.

### 2.2 Skenario Eksploitasi Konkuren (Concurrent Replay Window)
Misalkan token refresh `RT_0` aktif. Penyerang yang mencuri `RT_0` mengirimkan request refresh tepat bersamaan dengan request refresh sah dari browser pengguna:

```
Timeline:
T1: Request 1 (Pengguna) tiba di Usecase -> GetByHash(RT_0) -> revoked_at masih NULL -> LOLOS.
T2: Request 2 (Penyerang) tiba di Usecase -> GetByHash(RT_0) -> revoked_at masih NULL -> LOLOS.
T3: Request 1 masuk ke Rotate() -> Begin Tx -> SELECT FOR UPDATE (berhasil kunci baris RT_0).
T4: Request 2 masuk ke Rotate() -> Begin Tx -> SELECT FOR UPDATE (TERTAHAN / MENUNGGU LOCK Request 1).
T5: Request 1 melakukan INSERT RT_1, menyetel RT_0.revoked_at = NOW(), Commit Tx, dan melepaskan lock.
T6: Request 2 memperoleh lock RT_0. Query SELECT FOR UPDATE mengembalikan revoked_at = NOW().
T7: KARENA TIDAK ADA PENGECEKAN revokedAt != nil, Request 2 TETAP MENJALANKAN INSERT RT_2 dan COMMIT!
```

**Hasil Anomali**:
- Baik pengguna sah maupun penyerang **sama-sama menerima token aktif baru** (`RT_1` dan `RT_2`).
- Fitur pendeteksian penyalahgunaan token (*Refresh Token Reuse Detection* yang wajib mencabut semua token pengguna jika token lama digunakan kembali) **gagal terpicu**.
- Penyerang tetap mempertahankan akses persisten tanpa terdeteksi.

---

## 3. Tingkat Keparahan & Dampak
- **Tingkat Keparahan**: **MEDIUM-HIGH (CVSS: 6.8)**
- **Kelemahan Standar**: CWE-362 (Race Condition), CWE-384 (Session Fixation / Broken Replay Protection).
- **Dampak**: Kegagalan mekanisme keamanan zero-trust refresh token rotation; penyerang yang mencuri refresh token dapat mempertahankan sesi aktif berdampingan dengan pengguna asli.

---

## 4. Solusi & Rekomendasi Kode

### 4.1 Perbaikan pada `refresh_token_repo.go`
Tambahkan pengecekan eksplisit setelah row lock diperoleh. Jika `revokedAt != nil`, transaksi harus segera di-rollback dan mengembalikan sentinel error `ErrRefreshTokenRevoked`:

```go
var ErrRefreshTokenRevoked = errors.New("refresh token already revoked")

func (r *RefreshTokenRepo) Rotate(ctx context.Context, oldID uuid.UUID, newToken *domain.RefreshToken) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var revokedAt *time.Time
	if err := tx.QueryRow(ctx,
		`SELECT revoked_at FROM refresh_tokens WHERE id = $1 FOR UPDATE`, oldID,
	).Scan(&revokedAt); err != nil {
		return fmt.Errorf("failed to lock refresh token: %w", err)
	}

	// [PERBAIKAN]: Tolak jika token telah dicabut oleh request lain yang selesai mendahului
	if revokedAt != nil {
		return ErrRefreshTokenRevoked
	}

	if err := tx.QueryRow(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at, user_agent)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		newToken.UserID, newToken.TokenHash, newToken.ExpiresAt, newToken.UserAgent,
	).Scan(&newToken.ID, &newToken.CreatedAt); err != nil {
		return fmt.Errorf("failed to insert rotated token: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now(), replaced_by = $2, last_used_at = now() WHERE id = $1`,
		oldID, newToken.ID); err != nil {
		return fmt.Errorf("failed to revoke old token: %w", err)
	}

	return tx.Commit(ctx)
}
```

### 4.2 Penyesuaian pada `AuthUsecase.Refresh`
Jika `Rotate` mengembalikan `ErrRefreshTokenRevoked`, usecase harus mengeksekusi pencabutan massal (`RevokeAllForUser`) sebagai aksi pertahanan reuse detection:

```go
// backend/internal/usecase/auth_usecase.go:171
if err := u.refreshRepo.Rotate(ctx, existing.ID, newToken); err != nil {
    if errors.Is(err, domain.ErrRefreshTokenRevoked) {
        _ = u.refreshRepo.RevokeAllForUser(ctx, existing.UserID)
        return nil, ErrInvalidRefresh
    }
    return nil, err
}
```
