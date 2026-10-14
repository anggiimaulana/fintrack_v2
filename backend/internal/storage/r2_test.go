package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeS3 is a minimal in-process S3-compatible server: enough to exercise the
// R2 driver (PutObject/GetObject/DeleteObject/HeadBucket) without a real bucket.
type fakeS3 struct {
	mu      sync.Mutex
	objects map[string][]byte
	etags   map[string]string
}

func newFakeS3() *fakeS3 {
	return &fakeS3{objects: map[string][]byte{}, etags: map[string]string{}}
}

func (f *fakeS3) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Path style: /<bucket>/<key...>
		path := strings.TrimPrefix(r.URL.Path, "/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) < 2 || parts[0] == "" {
			// HeadBucket: /<bucket>
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusOK)
				return
			}
			http.NotFound(w, r)
			return
		}
		key := parts[1]

		f.mu.Lock()
		defer f.mu.Unlock()

		switch r.Method {
		case http.MethodHead:
			if _, ok := f.objects[key]; !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			sum := sha256.Sum256(body)
			etag := hex.EncodeToString(sum[:])
			f.objects[key] = body
			f.etags[key] = etag
			w.Header().Set("ETag", `"`+etag+`"`)
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			body, ok := f.objects[key]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`<?xml version="1.0"?><Error><Code>NoSuchKey</Code></Error>`))
				return
			}
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("Content-Length", itoa(len(body)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		case http.MethodDelete:
			delete(f.objects, key)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unsupported", http.StatusMethodNotAllowed)
		}
	})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func newTestR2(t *testing.T) (*R2Store, *fakeS3) {
	t.Helper()
	fake := newFakeS3()
	srv := httptest.NewServer(fake.handler())
	t.Cleanup(srv.Close)

	store, err := NewR2Store(context.Background(), R2Config{
		Endpoint:        srv.URL,
		Region:          "auto",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		Bucket:          "test-bucket",
		Prefix:          "fintrack",
	})
	if err != nil {
		t.Fatalf("NewR2Store: %v", err)
	}
	return store, fake
}

func TestR2Store_PutOpenDelete(t *testing.T) {
	ctx := context.Background()
	store, fake := newTestR2(t)

	key := ProofKey("pay-42")
	if err := store.Put(ctx, key, []byte("proof-bytes"), "image/png"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	// The object must land under the configured prefix.
	if _, ok := fake.objects["fintrack/payment-proofs/pay-42"]; !ok {
		t.Fatalf("object not stored under prefix; keys=%v", keysOf(fake.objects))
	}

	obj, err := store.Open(ctx, key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got, _ := io.ReadAll(obj.Body)
	_ = obj.Body.Close()
	if !bytes.Equal(got, []byte("proof-bytes")) {
		t.Fatalf("content = %q", got)
	}
	if obj.ContentType != "image/png" {
		t.Fatalf("content-type = %s", obj.ContentType)
	}

	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Open(ctx, key); err != ErrNotFound {
		t.Fatalf("Open after delete = %v, want ErrNotFound", err)
	}
}

func TestR2Store_Healthy(t *testing.T) {
	store, _ := newTestR2(t)
	if err := store.Healthy(context.Background()); err != nil {
		t.Fatalf("Healthy: %v", err)
	}
}

func TestR2Store_DisabledWithoutConfig(t *testing.T) {
	if _, err := NewR2Store(context.Background(), R2Config{Bucket: "x"}); err == nil {
		t.Fatal("expected error for incomplete config")
	}
}

// TestProvider_SwitchLocalToR2 proves the runtime switch: writes go to local,
// then after switching the setting, new writes land in R2 — and reads of the
// earlier local object still succeed (cross-driver fallback).
func TestProvider_SwitchLocalToR2(t *testing.T) {
	ctx := context.Background()
	local := newTestLocal(t)
	r2, fake := newTestR2(t)
	ms := &mutableSettings{val: DriverLocal}

	p := NewProvider(Config{Local: local, R2: r2, Settings: ms, CacheTTL: time.Hour})

	// 1) Active driver is local.
	st, err := p.Active(ctx)
	if err != nil || st.Name() != DriverLocal {
		t.Fatalf("active = %v (%v), want local", st, err)
	}
	oldKey := ProofKey("before-switch")
	if err := st.Put(ctx, oldKey, []byte("in-local"), "image/png"); err != nil {
		t.Fatalf("local put: %v", err)
	}

	// 2) Switch the setting to r2 and invalidate the cache.
	ms.val = DriverR2
	p.Invalidate()
	st2, err := p.Active(ctx)
	if err != nil {
		t.Fatalf("Active after switch: %v", err)
	}
	if st2.Name() != DriverR2 {
		t.Fatalf("active after switch = %s, want r2", st2.Name())
	}

	// 3) New writes land in R2.
	newKey := ProofKey("after-switch")
	if err := st2.Put(ctx, newKey, []byte("in-r2"), "image/png"); err != nil {
		t.Fatalf("r2 put: %v", err)
	}
	if _, ok := fake.objects["fintrack/payment-proofs/after-switch"]; !ok {
		t.Fatalf("new object not in r2; keys=%v", keysOf(fake.objects))
	}

	// 4) Reads still work for the object written before the switch (fallback).
	obj, err := p.Open(ctx, oldKey)
	if err != nil {
		t.Fatalf("Open old (cross-driver fallback): %v", err)
	}
	got, _ := io.ReadAll(obj.Body)
	_ = obj.Body.Close()
	if string(got) != "in-local" {
		t.Fatalf("old object content = %q, want in-local", got)
	}

	// 5) And for the new object.
	obj2, err := p.Open(ctx, newKey)
	if err != nil {
		t.Fatalf("Open new: %v", err)
	}
	got2, _ := io.ReadAll(obj2.Body)
	_ = obj2.Body.Close()
	if string(got2) != "in-r2" {
		t.Fatalf("new object content = %q, want in-r2", got2)
	}

	// 6) Status reflects r2 active + configured + healthy.
	status := p.Status(ctx)
	if status.Active != DriverR2 || !status.R2Configured || !status.R2Healthy {
		t.Fatalf("status = %+v", status)
	}
}

func keysOf(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
