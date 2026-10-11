# ✅ Tugas-Rutin — Checklist Recurring

Folder ini menyimpan checklist tugas rutin yang berulang, plus audit trail aktivitas.

## Isi
- `log_aktivitas.md` — audit trail aksi teknis (append-only, terbaru di atas).

## Template Checklist
Setiap periode buat file baru `checklist-YYYY-MM-DD.md` dari template berikut:

```markdown
# Checklist — YYYY-MM-DD

## Backend
- [ ] `cd backend && go vet ./...`
- [ ] `cd backend && go test ./...`
- [ ] `cd backend && go build ./...`
- [ ] 0 `context.Background()` di `internal/`

## Frontend
- [ ] `cd frontend && pnpm build`
- [ ] `cd frontend && pnpm lint`
- [ ] `cd frontend && pnpm test`
- [ ] 0 referensi `supabase` di `app/components/hooks/lib`

## Database
- [ ] Migrasi replayable dari nol (`fintrack_test`)
- [ ] Regenerate dump bila schema berubah (`database/scripts/dump.sh`)
- [ ] Backup drill (dump → restore → count identik)

## Umum
- [ ] Update `2_Projects/P01_FinTrack_v2/roadmap.md` (HUD)
- [ ] Triage `0_Inbox/`
- [ ] Tidak ada `.env`/secret ter-commit
```

## Aturan
1. **Append-only** untuk `log_aktivitas.md` — jangan edit entri lama.
2. Tugas yang selesai dicentang (`[x]`), bukan dihapus.
3. Checklist lama yang sudah selesai di-review → arsipkan ke `../../5_Archives/sprint-logs/`.
