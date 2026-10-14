package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrNotificationNotFound is returned when a notification does not exist or is
// not owned by the requesting user (the two cases are deliberately
// indistinguishable so a probe cannot reveal another user's ids).
var ErrNotificationNotFound = errors.New("notification not found")

// NotificationPreferences is a user's opt-in/opt-out for each email channel.
// The in-app inbox is always written regardless of these flags.
type NotificationPreferences struct {
	TransactionEmail bool `json:"transaction_email"`
	CategoryReminder bool `json:"category_reminder"`
	WeeklyReport     bool `json:"weekly_report"`
}

// DefaultNotificationPreferences matches the previous UI defaults so existing
// behaviour does not silently change for users who never touched the toggles.
func DefaultNotificationPreferences() NotificationPreferences {
	return NotificationPreferences{TransactionEmail: true, CategoryReminder: true, WeeklyReport: false}
}

// Notification is a durable in-app message for the header bell.
type Notification struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	Event     string     `json:"event"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Href      string     `json:"href"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// NotificationFilter drives the paged notification list (table page).
type NotificationFilter struct {
	Page    int
	PerPage int
	Sort    string // created_at | title | event
	Order   string // asc | desc
	Search  string // matches title or body
	Status  string // all | read | unread
}

// NotificationStats is the aggregate summary shown above the table.
type NotificationStats struct {
	Total  int `json:"total"`
	Unread int `json:"unread"`
	Read   int `json:"read"`
}

// NotificationRepository persists preferences and the in-app inbox.
type NotificationRepository interface {
	GetPreferences(ctx context.Context, userID uuid.UUID) (NotificationPreferences, error)
	SavePreferences(ctx context.Context, userID uuid.UUID, p NotificationPreferences) error
	// Create appends an in-app notification.
	Create(ctx context.Context, n *Notification) error
	// ListRecent returns the most recent notifications, newest first.
	ListRecent(ctx context.Context, userID uuid.UUID, limit int) ([]Notification, error)
	// ListPaged returns one page of notifications plus the total matching count.
	ListPaged(ctx context.Context, userID uuid.UUID, f NotificationFilter) ([]Notification, int, error)
	// Stats returns total/unread/read counts for the user.
	Stats(ctx context.Context, userID uuid.UUID) (NotificationStats, error)
	// GetByID returns one notification, scoped to its owner.
	GetByID(ctx context.Context, userID, id uuid.UUID) (*Notification, error)
	UnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	MarkRead(ctx context.Context, userID, id uuid.UUID) error
	// MarkReadMany marks the given ids read (no-op for ids not owned by the user).
	MarkReadMany(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
	// Delete removes one notification owned by the user.
	Delete(ctx context.Context, userID, id uuid.UUID) error
	// DeleteMany removes the given ids owned by the user.
	DeleteMany(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) error
	Clear(ctx context.Context, userID uuid.UUID) error
}
