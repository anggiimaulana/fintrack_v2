package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// memSettings is a stub SettingsReader.
type memSettings struct {
	val string
	err error
}

func (m memSettings) GetString(context.Context, string) (string, error) { return m.val, m.err }

func newTestLocal(t *testing.T) *LocalStore {
	t.Helper()
	dir := t.TempDir()
	ls, err := NewLocalStore(dir)
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	return ls
}

func TestLocalStore_PutOpenDelete(t *testing.T) {
	ctx := context.Background()
	ls := newTestLocal(t)

	key := ProofKey("abc-123")
	if err := ls.Put(ctx, key, []byte("hello"), "text/plain"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	obj, err := ls.Open(ctx, key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got, _ := io.ReadAll(obj.Body)
	_ = obj.Body.Close()
	if string(got) != "hello" {
		t.Fatalf("content = %q, want hello", got)
	}

	if err := ls.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := ls.Open(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Open after delete = %v, want ErrNotFound", err)
	}
	// Deleting a missing object is not an error.
	if err := ls.Delete(ctx, key); err != nil {
		t.Fatalf("Delete missing = %v, want nil", err)
	}
}

func TestLocalStore_PathTraversalRejected(t *testing.T) {
	ctx := context.Background()
	ls := newTestLocal(t)

	// A key that tries to escape the root must be sanitised or rejected.
	err := ls.Put(ctx, "../../etc/passwd", []byte("x"), "")
	if err != nil {
		t.Fatalf("Put sanitised key: %v", err)
	}
	// Whatever landed must be inside the root.
	root := ls.Root()
	matches, _ := filepath.Glob(filepath.Join(root, "**", "*"))
	for _, m := range matches {
		rel, _ := filepath.Rel(root, m)
		if len(rel) >= 2 && rel[0] == '.' && rel[1] == '.' {
			t.Fatalf("file escaped root: %s", m)
		}
	}
}

func TestLocalStore_Healthy(t *testing.T) {
	ctx := context.Background()
	ls := newTestLocal(t)
	if err := ls.Healthy(ctx); err != nil {
		t.Fatalf("Healthy = %v, want nil", err)
	}
}

// ─── Provider (runtime switch) ────────────────────────────────

func TestProvider_SwitchDriverFromSetting(t *testing.T) {
	ctx := context.Background()
	local := newTestLocal(t)

	// No r2 configured: active is always local even if the setting says r2.
	p := NewProvider(Config{Local: local, Settings: memSettings{val: DriverR2}})
	st, err := p.Active(ctx)
	if err != nil {
		t.Fatalf("Active: %v", err)
	}
	if st.Name() != DriverLocal {
		t.Fatalf("active = %s, want local (r2 not configured)", st.Name())
	}
}

func TestProvider_DefaultDriver(t *testing.T) {
	ctx := context.Background()
	local := newTestLocal(t)
	p := NewProvider(Config{DefaultDriver: DriverR2, Local: local})
	st, _ := p.Active(ctx)
	if st.Name() != DriverLocal {
		t.Fatalf("active = %s, want local fallback", st.Name())
	}
}

func TestProvider_SettingCachedThenInvalidated(t *testing.T) {
	ctx := context.Background()
	local := newTestLocal(t)
	// Use a mutable settings stub.
	ms := &mutableSettings{val: DriverLocal}
	p := NewProvider(Config{Local: local, Settings: ms, CacheTTL: time.Hour})

	if st, _ := p.Active(ctx); st.Name() != DriverLocal {
		t.Fatalf("active = %s, want local", st.Name())
	}
	// Change the setting; the cached value must still win until invalidated.
	ms.val = DriverR2
	if st, _ := p.Active(ctx); st.Name() != DriverLocal {
		t.Fatalf("cached active = %s, want local", st.Name())
	}
	p.Invalidate()
	// Still local because r2 is not configured in this provider.
	if st, _ := p.Active(ctx); st.Name() != DriverLocal {
		t.Fatalf("active = %s, want local", st.Name())
	}
}

func TestProvider_OpenFallsBackAcrossDrivers(t *testing.T) {
	ctx := context.Background()
	local := newTestLocal(t)

	// Put directly into the local store.
	key := ProofKey("pay-1")
	if err := local.Put(ctx, key, []byte("proof"), "image/png"); err != nil {
		t.Fatalf("local Put: %v", err)
	}

	// Provider with a fake "r2" primary that has nothing: Open must fall back to local.
	p := NewProvider(Config{Local: local, R2: nil, Settings: memSettings{val: DriverLocal}})
	obj, err := p.Open(ctx, key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer obj.Body.Close()
	got, _ := io.ReadAll(obj.Body)
	if string(got) != "proof" {
		t.Fatalf("content = %q, want proof", got)
	}
}

func TestProvider_StatusReportsLocal(t *testing.T) {
	ctx := context.Background()
	local := newTestLocal(t)
	p := NewProvider(Config{Local: local, Settings: memSettings{val: DriverLocal}})
	st := p.Status(ctx)
	if st.Active != DriverLocal || !st.LocalReady {
		t.Fatalf("status = %+v, want active=local ready=true", st)
	}
	if st.R2Configured {
		t.Fatalf("r2_configured = true, want false")
	}
}

// ─── helpers ──────────────────────────────────────────────────

type mutableSettings struct{ val string }

func (m *mutableSettings) GetString(context.Context, string) (string, error) { return m.val, nil }

func TestExtensionAndContentType(t *testing.T) {
	cases := []struct {
		mime, ext string
	}{
		{"image/png", ".png"},
		{"image/jpeg", ".jpg"},
		{"image/webp", ".webp"},
		{"application/pdf", ".bin"},
	}
	for _, c := range cases {
		if got := ExtensionFor(c.mime); got != c.ext {
			t.Errorf("ExtensionFor(%s) = %s, want %s", c.mime, got, c.ext)
		}
	}
	if got := ContentTypeFor("payment-proofs/x.png"); got != "image/png" {
		t.Errorf("ContentTypeFor png = %s", got)
	}
}

func TestProofKeySanitises(t *testing.T) {
	if got := ProofKey("../../evil"); got != "payment-proofs/evil" {
		t.Fatalf("ProofKey = %q", got)
	}
	if got := ProofKey(""); got != "payment-proofs/unknown" {
		t.Fatalf("ProofKey empty = %q", got)
	}
}

var _ = os.Getenv
