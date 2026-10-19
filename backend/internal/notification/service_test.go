package notification

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/pkg/contracts"
)

// stubRepo records created notifications and returns a fixed preference set.
type stubRepo struct {
	domain.NotificationRepository
	created []domain.Notification
	prefs   domain.NotificationPreferences
}

func (s *stubRepo) Create(_ context.Context, n *domain.Notification) (bool, error) {
	s.created = append(s.created, *n)
	return true, nil
}

func (s *stubRepo) GetPreferences(context.Context, uuid.UUID) (domain.NotificationPreferences, error) {
	return s.prefs, nil
}

// UnreadCount satisfies the interface; the emitter reads it to include the new
// badge count in the realtime payload.
func (s *stubRepo) UnreadCount(context.Context, uuid.UUID) (int, error) {
	return len(s.created), nil
}

// captureEmitter records published user events for assertions.
type captureEmitter struct {
	events []contracts.Event
	users  []uuid.UUID
}

func (c *captureEmitter) PublishUser(_ context.Context, userID uuid.UUID, ev contracts.Event) {
	c.users = append(c.users, userID)
	c.events = append(c.events, ev)
}

// TestNotify_EmitsCreatedEvent verifies a persisted notification is pushed to
// the owner in realtime, carrying the row so the client can patch in place.
func TestNotify_EmitsCreatedEvent(t *testing.T) {
	repo := &stubRepo{}
	em := &captureEmitter{}
	svc := NewService(repo, nil, nil).WithEmitter(em)

	uid := uuid.New()
	if err := svc.Notify(context.Background(), Message{
		UserID: uid, Event: contracts.EventPaymentReceived, Title: "T", Body: "B", Href: "/x",
	}); err != nil {
		t.Fatalf("notify: %v", err)
	}

	if len(em.events) != 1 {
		t.Fatalf("expected 1 emitted event, got %d", len(em.events))
	}
	if em.events[0].Event != contracts.EventNotificationCreated {
		t.Fatalf("expected %s, got %s", contracts.EventNotificationCreated, em.events[0].Event)
	}
	if em.users[0] != uid {
		t.Fatal("event emitted to the wrong user")
	}
}

// TestNotify_NilEmitterIsSafe verifies the service works without realtime wired.
func TestNotify_NilEmitterIsSafe(t *testing.T) {
	repo := &stubRepo{}
	svc := NewService(repo, nil, nil)
	if err := svc.Notify(context.Background(), Message{UserID: uuid.New(), Event: "x", Title: "t", Body: "b"}); err != nil {
		t.Fatalf("notify must not fail without an emitter: %v", err)
	}
	if len(repo.created) != 1 {
		t.Fatal("notification must still be persisted")
	}
}

// TestEmitter_AlwaysPersistsInApp verifies the in-app inbox is written even when
// the user muted the email channel — the fallback the spec promises.
func TestEmitter_AlwaysPersistsInApp(t *testing.T) {
	repo := &stubRepo{prefs: domain.NotificationPreferences{TransactionEmail: false}}
	svc := NewService(repo, nil, nil) // nil mailer -> email path skipped
	em := NewEmitter(nil, svc)

	uid := uuid.New()
	em.PublishUser(context.Background(), uid, contracts.NewEvent(contracts.EventPaymentReceived, 0, contracts.PaymentReceivedPayload{
		AmountIDR: 39000,
	}))

	if len(repo.created) != 1 {
		t.Fatalf("expected 1 persisted notification, got %d", len(repo.created))
	}
	if repo.created[0].UserID != uid {
		t.Fatal("notification persisted for the wrong user")
	}
}

// TestEmitter_IgnoresAdminEvents verifies admin-only events never reach a user's
// inbox (they are operational, not personal).
func TestEmitter_IgnoresAdminEvents(t *testing.T) {
	repo := &stubRepo{}
	svc := NewService(repo, nil, nil)
	em := NewEmitter(nil, svc)

	em.PublishUser(context.Background(), uuid.New(), contracts.NewEvent(contracts.EventAIEndpointDown, 0, nil))

	if len(repo.created) != 0 {
		t.Fatalf("admin event must not create a user notification, got %d", len(repo.created))
	}
}

// TestNotify_DedupedMessageIsSilent verifies a duplicate (dedupe key already
// stored) neither emits realtime nor reaches email.
func TestNotify_DedupedMessageIsSilent(t *testing.T) {
	em := &captureEmitter{}
	svc := NewService(&droppedRepo{}, nil, nil).WithEmitter(em)
	if err := svc.Notify(context.Background(), Message{
		UserID: uuid.New(), Event: "x", Title: "T", Body: "B", DedupeKey: "k",
	}); err != nil {
		t.Fatalf("notify: %v", err)
	}
	if len(em.events) != 0 {
		t.Fatalf("duplicate must not emit, got %d events", len(em.events))
	}
}

// droppedRepo reports every insert as a duplicate.
type droppedRepo struct{ domain.NotificationRepository }

func (d *droppedRepo) Create(context.Context, *domain.Notification) (bool, error) { return false, nil }

func TestAllowedByPreferences(t *testing.T) {
	prefs := domain.NotificationPreferences{TransactionEmail: true, CategoryReminder: false, WeeklyReport: false}
	if !allowedByPreferences(prefs, ChannelTransaction) {
		t.Error("transaction channel should be allowed when the flag is on")
	}
	if allowedByPreferences(prefs, ChannelCategory) {
		t.Error("category channel must be blocked when the flag is off")
	}
	if allowedByPreferences(prefs, Channel("unknown")) {
		t.Error("unknown channel must fail closed")
	}
}
