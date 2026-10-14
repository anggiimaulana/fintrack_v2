// Package notification turns domain events into durable in-app notifications and
// (when configured and permitted by the user's preferences) email.
package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/mail"
	"github.com/fintrack/backend/pkg/contracts"
	"github.com/fintrack/backend/pkg/logger"
)

// Channel selects which preference gates an email. The in-app inbox is always
// written, so a user who muted a channel still sees the event in the bell.
type Channel string

const (
	ChannelTransaction Channel = "transaction_email"
	ChannelCategory    Channel = "category_reminder"
	ChannelWeekly      Channel = "weekly_report"
)

// Message is a notification to deliver.
type Message struct {
	UserID  uuid.UUID
	Event   string
	Title   string
	Body    string
	Href    string
	Channel Channel
}

// EmailResolver returns the recipient address for a user. Kept as a function so
// the service does not depend on the user repository.
type EmailResolver func(ctx context.Context, userID uuid.UUID) (string, error)

// RealtimeEmitter publishes a per-user event. Declared narrowly so this package
// does not depend on the concrete transport. Nil is safe (no-op).
type RealtimeEmitter interface {
	PublishUser(ctx context.Context, userID uuid.UUID, ev contracts.Event)
}

// Service persists notifications and delivers email when permitted.
type Service struct {
	repo    domain.NotificationRepository
	mailer  *mail.Provider
	resolve EmailResolver
	emitter RealtimeEmitter
}

func NewService(repo domain.NotificationRepository, mailer *mail.Provider, resolve EmailResolver) *Service {
	return &Service{repo: repo, mailer: mailer, resolve: resolve}
}

// WithEmitter installs the realtime emitter used to push per-item notification
// events to the owning user (so the bell and the notifications page update
// without a full refetch). Nil is accepted and disables emission.
func (s *Service) WithEmitter(e RealtimeEmitter) *Service {
	s.emitter = e
	return s
}

// emit publishes a per-user notification event, best-effort. Realtime is an
// accelerator, never the source of truth, so a publish failure is ignored.
func (s *Service) emit(ctx context.Context, userID uuid.UUID, name string, payload contracts.NotificationChangedPayload) {
	if s.emitter == nil {
		return
	}
	s.emitter.PublishUser(ctx, userID, contracts.NewEvent(name, time.Now().Unix(), payload))
}

// EmitNotification lets the HTTP handler publish inbox lifecycle events (read /
// deleted) that do not go through Notify. Best-effort and nil-safe.
func (s *Service) EmitNotification(ctx context.Context, userID uuid.UUID, name string, payload contracts.NotificationChangedPayload) {
	s.emit(ctx, userID, name, payload)
}

// unreadCount reads the user's unread count, returning 0 on error (the badge is
// non-critical; a stale count self-heals on the next fetch).
func (s *Service) unreadCount(ctx context.Context, userID uuid.UUID) int {
	n, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return 0
	}
	return n
}

// toContractNotification maps a domain notification to its wire form.
func toContractNotification(n *domain.Notification) *contracts.Notification {
	var readAt *string
	if n.ReadAt != nil {
		s := n.ReadAt.UTC().Format(time.RFC3339)
		readAt = &s
	}
	return &contracts.Notification{
		ID:        n.ID.String(),
		Event:     n.Event,
		Title:     n.Title,
		Body:      n.Body,
		Href:      n.Href,
		ReadAt:    readAt,
		CreatedAt: n.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// Notify records the in-app notification and, when the user's preference for the
// message's channel is on and email is configured, sends an email.
//
// Delivery is best-effort: a mail failure never fails the caller's business
// operation, and the in-app entry (already committed) remains as the fallback.
func (s *Service) Notify(ctx context.Context, m Message) error {
	n := &domain.Notification{
		UserID: m.UserID,
		Event:  m.Event,
		Title:  m.Title,
		Body:   m.Body,
		Href:   m.Href,
	}
	if err := s.repo.Create(ctx, n); err != nil {
		return fmt.Errorf("persist notification: %w", err)
	}

	// Push the new row to the owner so the bell badge and the notifications page
	// update without a refetch.
	s.emit(ctx, n.UserID, contracts.EventNotificationCreated, contracts.NotificationChangedPayload{
		IDs:    []string{n.ID.String()},
		Item:   toContractNotification(n),
		Unread: s.unreadCount(ctx, n.UserID),
	})

	if s.mailer == nil || s.resolve == nil {
		return nil
	}
	sender := s.mailer.Sender(ctx)
	if !sender.Configured() {
		// Email not configured: the in-app entry is the fallback. Log once at
		// debug level so operators can see why no mail went out.
		logger.Debug("notification: email skipped (not configured)",
			zap.String("event", m.Event), zap.String("user_id", m.UserID.String()))
		return nil
	}

	prefs, err := s.repo.GetPreferences(ctx, m.UserID)
	if err != nil {
		logger.Warn("notification: could not read preferences; skipping email",
			zap.Error(err), zap.String("user_id", m.UserID.String()))
		return nil
	}
	if !allowedByPreferences(prefs, m.Channel) {
		return nil
	}

	to, err := s.resolve(ctx, m.UserID)
	if err != nil || to == "" {
		logger.Warn("notification: no email address; skipping email",
			zap.String("user_id", m.UserID.String()))
		return nil
	}

	if err := sender.Send(ctx, to, m.Title, m.Body); err != nil {
		if errors.Is(err, mail.ErrDisabled) {
			return nil
		}
		// Best-effort: the in-app notification is the fallback, so we only log.
		logger.Error("notification: email send failed",
			zap.Error(err), zap.String("event", m.Event), zap.String("user_id", m.UserID.String()))
	}
	return nil
}

// allowedByPreferences maps a channel to the governing preference flag.
func allowedByPreferences(p domain.NotificationPreferences, ch Channel) bool {
	switch ch {
	case ChannelTransaction:
		return p.TransactionEmail
	case ChannelCategory:
		return p.CategoryReminder
	case ChannelWeekly:
		return p.WeeklyReport
	default:
		// Unknown channel: do not send. Fail closed so a new event type does not
		// silently start emailing users who never opted in.
		return false
	}
}

// SendTestEmail sends a one-off test message, bypassing preferences. Used by the
// Super Admin "send test email" button to validate SMTP configuration.
func (s *Service) SendTestEmail(ctx context.Context, to string) error {
	if s.mailer == nil {
		return mail.ErrDisabled
	}
	sender := s.mailer.Sender(ctx)
	if !sender.Configured() {
		return mail.ErrDisabled
	}
	return sender.Send(ctx, to, "FinTrack — Tes Konfigurasi Email",
		"Ini adalah email uji dari FinTrack. Jika Anda menerimanya, konfigurasi SMTP sudah benar.")
}
