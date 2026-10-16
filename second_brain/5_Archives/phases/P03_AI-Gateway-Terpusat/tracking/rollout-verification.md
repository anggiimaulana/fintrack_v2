# Rollout Verification — P03 AI Gateway Terpusat

Bukti verifikasi TASK-P03-014 (benchmark & zero-downtime rollout).

## 1. Feature Flag

- **Flag:** `feature_ai_gateway_centralized` (tabel `feature_flags`).
- **Default:** OFF (fail-closed). Saat OFF, `AIUsecase` memakai jalur BYOK lama (tidak ada perubahan perilaku).
- **Saat ON:** `AIUsecase.WithGateway(gateway)` aktif → chat/vision/voice/embedding lewat pool terpusat.
- **Rollout tanpa deploy:** ubah `is_enabled`/`rollout` di DB → berlaku pada request berikutnya (cache flag 30s).

## 2. Benchmark (hasil nyata)

Perintah: `go test ./internal/aigateway/gateway/... -bench . -benchtime=300x -run "^$"`

```
BenchmarkChatStream-12      300   1183645 ns/op   110183 B/op   224 allocs/op
BenchmarkChatSync-12        300    968223 ns/op    38765 B/op   209 allocs/op
```

- Overhead transport gateway: **~1 ms** per operasi (di lingkungan lokal, termasuk mock upstream).
- Kriteria rencana ("overhead < 15ms") **terpenuhi**.

## 3. Uji Konkurensi (indikasi bebas kebocoran)

- `TestConcurrentStreamsNoLeak`: **50 stream SSE paralel** selesai tanpa hang (timeout 10s) → tidak ada kebocoran goroutine/koneksi pada jalur SSE.
- Catatan: uji dengan `-race` **tidak tersedia** di lingkungan ini (tidak ada gcc/cgo). Uji race harus dijalankan di CI dengan toolchain lengkap.

## 4. Zero-Downtime Rollout (prosedur)

1. Deploy backend baru dengan flag **OFF** (perilaku identik dengan P01) — tidak ada dampak user.
2. Pantau `/metrics` & `/health`.
3. Aktifkan flag untuk sebagian user (`rollout` %) → pantau error rate gateway & latensi.
4. Naikkan rollout bertahap → 100%.
5. Rollback = set flag OFF (instan, tanpa deploy).

## 5. Health Checker

- Worker aktif dengan interval **60s**, timeout probe **5s** (probe `GET {base_url}/models`).
- Endpoint mati ditandai `unhealthy`; pulih ditandai `healthy` otomatis.
- Diuji: `gateway/health_test.go` (4 test PASS).

## 6. Kesimpulan

| Kriteria | Status |
|---|---|
| Trafik AI lewat gateway saat flag ON | ✅ (adapter wired, diuji) |
| Failover < 2s (rencana) | ✅ < 1s (diuji) |
| Breaker 5 error → Open, cooldown 30s | ✅ (diuji) |
| Embedding 768d konsisten | ✅ (diuji, lock) |
| Token usage tercatat | ✅ (metering diuji) |
| Benchmark overhead < 15ms | ✅ ~1ms |
| Zero-downtime via flag | ✅ prosedur siap |
