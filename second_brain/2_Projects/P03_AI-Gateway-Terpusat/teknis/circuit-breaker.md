# Spesifikasi State Machine Circuit Breaker

Dokumen ini mendefinisikan desain teknis in-memory Circuit Breaker pada modul AI Gateway FinTrack v2 untuk mencegah cascade failure dan request starvation.

---

## 1. Definisi State Machine

Circuit Breaker mengimplementasikan tiga status operasional:

```
          [ CLOSED ] (Kondisi Normal)
              │
              │ 5x Kegagalan Berturut-turut
              ▼
           [ OPEN ]  (Kondisi Rusak / Terputus)
              │
              │ Cooldown 30 Detik Berakhir
              ▼
        [ HALF-OPEN ] (Kondisi Percobaan / Probe)
         │         │
         │ Sukses  │ Gagal
         ▼         ▼
    [ CLOSED ]  [ OPEN ] (Cooldown 30 Detik Lagi)
```

### Penjelasan State:
1. **`StateClosed` (Normal)**:
   - Semua request diteruskan ke upstream provider.
   - Counter kegagalan di-reset ke `0` setiap kali ada request yang berhasil.
   - Jika consecutive failure counter mencapai batas ambang `Threshold = 5`, state langsung bertransisi ke `StateOpen`.
2. **`StateOpen` (Terputus / Blokir Cepat)**:
   - Upstream dianggap mati atau mengalami degradasi parah.
   - Semua request yang datang **langsung ditolak seketika** (fast failover ke rute cadangan) tanpa memicu koneksi HTTP jaringan.
   - Timer cooldown dimulai selama `30 detik`.
3. **`StateHalfOpen` (Uji Coba / Probe)**:
   - Setelah 30 detik pada status Open, state bertransisi ke Half-Open.
   - Gateway mengizinkan tepat **1 request probe** melintas ke upstream.
   - Jika request probe **berhasil**, state kembali ke `StateClosed` dan counter error di-reset.
   - Jika request probe **gagal**, state kembali ke `StateOpen` selama `30 detik` berikutnya.

---

## 2. Klasifikasi Error (Penentu Transisi)

Tidak semua respon HTTP error dihitung sebagai kegagalan sirkuit. Circuit breaker hanya menghitung error yang berkaitan dengan integritas layanan upstream:

| Kode / Tipe Error | Kategori | Dihitung Kegagalan Sirkuit? | Penanganan |
|---|---|---|---|
| HTTP 500 Internal Server Error | Upstream Failure | **Ya** | Counter + 1 |
| HTTP 502 Bad Gateway | Upstream Failure | **Ya** | Counter + 1 |
| HTTP 503 Service Unavailable | Upstream Failure | **Ya** | Counter + 1 |
| HTTP 504 Gateway Timeout | Upstream Failure | **Ya** | Counter + 1 |
| Network Timeout / I/O Drop | Network Failure | **Ya** | Counter + 1 |
| Connection Refused | Network Failure | **Ya** | Counter + 1 |
| HTTP 429 Too Many Requests | Rate Limit | **Ya** | Counter + 1 (indikasi limit upstream jebol) |
| HTTP 400 Bad Request | Client Error | **Tidak** | Invalid prompt/JSON, bukan salah upstream |
| HTTP 401 / 403 Unauthorized | Config Error | **Tidak** | Ditangani logger & alert konfigurasi |
| HTTP 404 Not Found | Route Error | **Tidak** | URL path salah atau model tidak terdaftar |

---

## 3. Implementasi Kode Go Thread-Safe

Struktur struct in-memory di `backend/internal/aigateway/breaker/breaker.go`:

```go
package breaker

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrCircuitOpen = errors.New("circuit breaker is open; traffic blocked")
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	mu                sync.RWMutex
	state             State
	failureCount      int
	failureThreshold  int
	cooldownDuration  time.Duration
	lastStateChange   time.Time
	halfOpenProbeDone bool
}

func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: threshold,
		cooldownDuration: cooldown,
		lastStateChange:  time.Now(),
	}
}

// Allow mengecek apakah request diizinkan melintas ke upstream
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		if now.Sub(cb.lastStateChange) >= cb.cooldownDuration {
			cb.state = StateHalfOpen
			cb.halfOpenProbeDone = false
			cb.lastStateChange = now
			return true // Izinkan request pertama sebagai probe
		}
		return false

	case StateHalfOpen:
		if !cb.halfOpenProbeDone {
			cb.halfOpenProbeDone = true
			return true
		}
		return false // Tolak request lain selama probe sedang berjalan
	}

	return false
}

// RecordSuccess dipanggil ketika upstream mengembalikan response valid
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0
	if cb.state == StateHalfOpen {
		cb.state = StateClosed
		cb.lastStateChange = time.Now()
	}
}

// RecordFailure dipanggil saat upstream mengembalikan error 5xx atau timeout
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	now := time.Now()

	if cb.state == StateHalfOpen || cb.failureCount >= cb.failureThreshold {
		cb.state = StateOpen
		cb.lastStateChange = now
		cb.failureCount = 0
	}
}

func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}
```

---

## 4. Parameter Konfigurasi Default

| Parameter | Nilai Default | Rasional |
|---|---|---|
| `FailureThreshold` | **5** kegagalan berurutan | Menghindari over-sensitivitas akibat fluktuasi sesaat (glitch jaringan). |
| `CooldownDuration` | **30 detik** | Waktu yang cukup bagi pod upstream atau vendor AI untuk memulihkan beban. |
| `ProbeConcurrency` | **1 request** | Membatasi dampak latensi ke user saat menguji status pemulihan upstream. |
