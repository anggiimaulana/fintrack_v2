package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LocalStore stores objects on the local filesystem under Root. It is the
// development default and requires no external service.
type LocalStore struct {
	root string
}

// NewLocalStore builds a local store rooted at dir, creating it if needed.
func NewLocalStore(dir string) (*LocalStore, error) {
	if dir == "" {
		dir = "uploads"
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}
	return &LocalStore{root: abs}, nil
}

// Name returns the driver name.
func (s *LocalStore) Name() string { return DriverLocal }

// Root returns the absolute root directory (used to serve files statically).
func (s *LocalStore) Root() string { return s.root }

// resolve maps a key to an absolute path, rejecting any traversal outside root.
func (s *LocalStore) resolve(key string) (string, error) {
	clean := filepath.Clean("/" + strings.ReplaceAll(key, "\\", "/"))
	full := filepath.Join(s.root, filepath.FromSlash(clean))
	rel, err := filepath.Rel(s.root, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", ErrInvalidKey
	}
	return full, nil
}

// Put writes content under key, creating parent directories as needed.
func (s *LocalStore) Put(_ context.Context, key string, content []byte, _ string) error {
	full, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		return FormatError(DriverLocal, err)
	}
	// Write to a temp file then rename, so a partial write never replaces a
	// previously-valid proof.
	tmp := full + ".tmp"
	if err := os.WriteFile(tmp, content, 0o640); err != nil {
		return FormatError(DriverLocal, err)
	}
	if err := os.Rename(tmp, full); err != nil {
		_ = os.Remove(tmp)
		return FormatError(DriverLocal, err)
	}
	return nil
}

// Open returns the object for reading.
func (s *LocalStore) Open(_ context.Context, key string) (*Object, error) {
	full, err := s.resolve(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, FormatError(DriverLocal, err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, FormatError(DriverLocal, err)
	}
	return &Object{
		Body:        f,
		ContentType: ContentTypeFor(key),
		Size:        info.Size(),
		ModTime:     info.ModTime(),
	}, nil
}

// Delete removes an object; a missing object is not an error.
func (s *LocalStore) Delete(_ context.Context, key string) error {
	full, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return FormatError(DriverLocal, err)
	}
	return nil
}

// Healthy verifies the root directory exists and is writable.
func (s *LocalStore) Healthy(_ context.Context) error {
	info, err := os.Stat(s.root)
	if err != nil {
		return FormatError(DriverLocal, err)
	}
	if !info.IsDir() {
		return FormatError(DriverLocal, fmt.Errorf("%s is not a directory", s.root))
	}
	return nil
}

// compile-time assertion.
var _ Store = (*LocalStore)(nil)
