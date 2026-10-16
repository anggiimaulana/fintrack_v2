# Skenario Pengujian — P01 FinTrack v2

Matriks validasi. Semua bukti output disimpan di `tester/` (tidak pernah di-commit).

## 1. DB replayable
| # | Skenario | Perintah | Kriteria |
|---|----------|----------|----------|
| 1.1 | Migrasi dari nol | `database/scripts/migrate.sh` ke `fintrack_test` | sukses tanpa error |
| 1.2 | Struktur tabel | `\dt` | 14 tabel |
| 1.3 | Constraint kondisional | `\d transactions` | 3 constraint (income/expense/savings) |

## 2. Backend build + test
| # | Skenario | Perintah | Kriteria |
|---|----------|----------|----------|
| 2.1 | Vet/build/test | `cd backend && go vet ./... && go build ./... && go test ./...` | semua ok |
| 2.2 | Tanpa context.Background | `grep -rn "context.Background()" internal` | 0 hasil |

## 3. Integrasi auth + isolasi
`TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5435/fintrack_test go test ./internal/repository/postgres/ -run Integration -v`
| # | Skenario | Kriteria |
|---|----------|----------|
| 3.1 | register→login→refresh→reuse | reuse refresh → 401 + semua token revoked |
| 3.2 | Isolasi user | A akses transaksi B → 404 |
| 3.3 | Optimistic lock | update stale → 409 |
| 3.4 | Race SPK | 20 create paralel → baris baru ≤2, tanpa error |

## 4. Migrasi data
| # | Skenario | Kriteria |
|---|----------|----------|
| 4.1 | `03_verify.sql` lama vs baru | count & `sum(amount)` per user identik |
| 4.2 | `verify_output.txt` | tersimpan |

## 5. Login kredensial migrasi
`curl -i -X POST localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"budi@example.com","password":"User@123456"}'`
| # | Skenario | Kriteria |
|---|----------|----------|
| 5.1 | budi | 200 + `Set-Cookie: ft_at=...; HttpOnly; SameSite=Lax` |
| 5.2 | admin | 200 |

## 6. Isolasi via API
Login budi → catat `transaction.id`; login siti → `GET /api/v1/transactions/<id-budi>` → **404**.

## 7. Refresh rotation
Pakai `ft_rt` lama setelah 1x refresh → **401** + `refresh_tokens.revoked_at` terisi.

## 8. Keamanan
| # | Skenario | Kriteria |
|---|----------|----------|
| 8.1 | `Origin: http://evil.com` pada login | 403 |
| 8.2 | 6x login salah | 429 |
| 8.3 | `document.cookie` | cookie tidak terlihat (HttpOnly) |

## 9. RAG hidup
Buat transaksi "makan siang sate padang 35 ribu" → `count(transaction_embeddings)` +1; chat "berapa pengeluaran makan saya?" → balasan memakai data tsb.

## 10. Race SPK
`seq 20 | xargs -P20 curl ...` → `count(spk_results)` +≤2, server tanpa error log.

## 11. Cache benar
Dua user login, buka dashboard bergantian cepat → angka berbeda; setelah create transaksi → dashboard langsung berubah.

## 12. Performa (k6)
`docker run --rm -i grafana/k6 run - < tester/k6-smoke.js` (login + dashboard + list, 50 VU, 30s) → p95 < 300ms, checks > 99%; hasil ke `tester/k6-result.txt`.

## 13. Backup drill
`dump.sh` → `restore.sh` ke `fintrack_restore_test` → count sama → drop scratch.

## 14. Frontend
`cd frontend && pnpm build && pnpm lint && pnpm test` hijau; `grep -rn supabase frontend/{app,components,hooks,lib}` → 0.

## 15. UI walkthrough (375/768/1440 + dark mode)
Halaman: `/`, `/login`, `/register`, `/user`, `/user/transactions`, `/user/ai`, `/user/api-keys`, `/user/reports`, `/user/sessions`, `/user/settings`, `/user/profile`, `/user/voice`, `/admin`, `/admin/users`, `/admin/categories/{income,expense,savings}`, `/admin/categories/expense/[id]/subcategories`, `/admin/providers`.
Kriteria: tidak ada hex hardcoded (kecuali palette kategori), Rupiah `font-mono`, semua input berlabel, tabel scroll di 375px, empty/error state tampil saat query gagal, toggle dark bekerja & tersimpan.

## 16. Compose end-to-end
`docker compose up --build -d` → health 200 → login browser `localhost:3001` → buat transaksi → dashboard tampil → chat AI SSE streaming (bukan buffered).
