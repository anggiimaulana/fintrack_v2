// Package storage provides a backend-agnostic object store for user uploads
// (currently payment proof images). It abstracts two drivers:
//
//   - local: files on disk, served back through the API (development default).
//   - r2:    Cloudflare R2 (S3-compatible) object storage (production).
//
// The active driver is chosen at runtime from the `storage_driver` setting, so
// switching local <-> r2 takes effect immediately for new uploads and reads,
// without a restart. Objects written by one driver are still readable while
// that driver is active; switching drivers only affects which store new objects
// land in (a migration/re-sync is an operational concern, not a code one).
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// Driver names.
const (
	DriverLocal = "local"
	DriverR2    = "r2"
)

// Errors returned by the storage layer.
var (
	// ErrNotFound is returned when an object does not exist.
	ErrNotFound = errors.New("storage: object not found")
	// ErrDisabled is returned when the active driver is not configured.
	ErrDisabled = errors.New("storage: driver not configured")
	// ErrInvalidKey is returned for keys that escape the configured prefix.
	ErrInvalidKey = errors.New("storage: invalid key")
)

// Object is a stored blob plus the metadata callers need to serve it.
type Object struct {
	Body        io.ReadCloser
	ContentType string
	Size        int64
	// ModTime is the last-modified time (zero when unknown).
	ModTime time.Time
}

// Store is the object-store contract shared by every driver.
type Store interface {
	// Name returns the driver name ("local" | "r2").
	Name() string
	// Put stores content under key. contentType may be empty.
	Put(ctx context.Context, key string, content []byte, contentType string) error
	// Open returns the object for reading, or ErrNotFound.
	Open(ctx context.Context, key string) (*Object, error)
	// Delete removes an object (no error when it is already absent).
	Delete(ctx context.Context, key string) error
	// Healthy reports whether the driver is reachable/configured.
	Healthy(ctx context.Context) error
}

// ─── Key helpers ──────────────────────────────────────────────

// ProofKey builds the canonical object key for a payment proof. The key is
// deterministic per payment id, so re-uploading overwrites the previous proof
// instead of leaking orphaned objects.
func ProofKey(paymentID string) string {
	return "payment-proofs/" + sanitizeSegment(paymentID)
}

// AvatarKey builds the canonical object key for a user's avatar from a
// "<uuid>.<ext>" filename. The key is deterministic per user id, so re-uploading
// overwrites the previous avatar instead of leaking orphaned objects.
func AvatarKey(filename string) string {
	return "avatars/" + sanitizeSegment(filename)
}

// ReceiptKey builds the canonical object key for an uploaded receipt image.
func ReceiptKey(filename string) string {
	return "receipts/" + sanitizeSegment(filename)
}

// sanitizeSegment strips path separators and traversal sequences from a single
// key segment so a hostile id cannot escape the store prefix. A file extension
// (the part after the final dot) is preserved so serving can infer the type.
func sanitizeSegment(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\\", "")
	s = strings.ReplaceAll(s, "/", "")
	// Keep only the last dot so "a.b.png" -> "ab.png" stays a single segment
	// with a usable extension, and "..png" cannot become a traversal.
	for strings.Contains(s, "..") {
		s = strings.ReplaceAll(s, "..", ".")
	}
	// A leading dot would create a hidden file and is never part of a real id.
	s = strings.TrimLeft(s, ".")
	if s == "" {
		return "unknown"
	}
	return s
}

// ExtensionFor maps a detected image MIME type to a safe file extension.
func ExtensionFor(mime string) string {
	switch mime {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".bin"
	}
}

// ContentTypeFor maps a file extension back to a content type for serving.
func ContentTypeFor(key string) string {
	switch {
	case strings.HasSuffix(key, ".png"):
		return "image/png"
	case strings.HasSuffix(key, ".jpg"), strings.HasSuffix(key, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(key, ".webp"):
		return "image/webp"
	case strings.HasSuffix(key, ".gif"):
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}

// FormatError wraps an error with the driver name for clearer logs.
func FormatError(driver string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("storage[%s]: %w", driver, err)
}
