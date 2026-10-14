package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
)

// NotificationRepo persists notification preferences and the in-app inbox.
type NotificationRepo struct {
	pool *pgxpool.Pool
}

func NewNotificationRepo(pool *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{pool: pool}
}

// GetPreferences returns the user's preferences, materialising the defaults
// (without writing) when no row exists yet.
func (r *NotificationRepo) GetPreferences(ctx context.Context, userID uuid.UUID) (domain.NotificationPreferences, error) {
	var p domain.NotificationPreferences
	err := r.pool.QueryRow(ctx, `
		SELECT transaction_email, category_reminder, weekly_report
		FROM notification_preferences WHERE user_id = $1`, userID).
		Scan(&p.TransactionEmail, &p.CategoryReminder, &p.WeeklyReport)
	if err != nil {
		// No row yet (or read error): fall back to defaults rather than failing
		// the whole settings screen.
		return domain.DefaultNotificationPreferences(), nil
	}
	return p, nil
}

// SavePreferences upserts the user's preferences.
func (r *NotificationRepo) SavePreferences(ctx context.Context, userID uuid.UUID, p domain.NotificationPreferences) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO notification_preferences (user_id, transaction_email, category_reminder, weekly_report, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (user_id) DO UPDATE SET
			transaction_email = EXCLUDED.transaction_email,
			category_reminder = EXCLUDED.category_reminder,
			weekly_report     = EXCLUDED.weekly_report,
			updated_at        = now()`,
		userID, p.TransactionEmail, p.CategoryReminder, p.WeeklyReport)
	if err != nil {
		return fmt.Errorf("save notification preferences: %w", err)
	}
	return nil
}

func (r *NotificationRepo) Create(ctx context.Context, n *domain.Notification) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, event, title, body, href)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`,
		n.UserID, n.Event, n.Title, n.Body, n.Href).Scan(&n.ID, &n.CreatedAt)
}

func (r *NotificationRepo) ListRecent(ctx context.Context, userID uuid.UUID, limit int) ([]domain.Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, event, title, body, href, read_at, created_at
		FROM notifications WHERE user_id = $1
		ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	out := []domain.Notification{}
	for rows.Next() {
		var n domain.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Event, &n.Title, &n.Body, &n.Href, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		out = append(out, n)
	}
	return out, nil
}

func (r *NotificationRepo) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

// allowedNotificationSorts is the ORDER BY whitelist. Never interpolate user
// input into SQL directly; only these exact strings reach the query.
var allowedNotificationSorts = map[string]string{
	"created_at": "created_at",
	"title":      "title",
	"event":      "event",
}

// ListPaged returns one page of the user's notifications plus the total count of
// rows matching the filter. Sorting is whitelisted, search uses ILIKE over
// title/body, and status filters on read_at.
func (r *NotificationRepo) ListPaged(ctx context.Context, userID uuid.UUID, f domain.NotificationFilter) ([]domain.Notification, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 100 {
		f.PerPage = 20
	}
	orderCol, ok := allowedNotificationSorts[f.Sort]
	if !ok {
		orderCol = "created_at"
	}
	orderDir := "DESC"
	if strings.EqualFold(f.Order, "asc") {
		orderDir = "ASC"
	}

	conditions := []string{"user_id = $1"}
	args := []interface{}{userID}
	argIdx := 2

	if f.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR body ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+f.Search+"%")
		argIdx++
	}
	switch f.Status {
	case "read":
		conditions = append(conditions, "read_at IS NOT NULL")
	case "unread":
		conditions = append(conditions, "read_at IS NULL")
	}

	where := strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM notifications WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	offset := (f.Page - 1) * f.PerPage
	query := fmt.Sprintf(`
		SELECT id, user_id, event, title, body, href, read_at, created_at
		FROM notifications WHERE %s
		ORDER BY %s %s, id DESC
		LIMIT $%d OFFSET $%d`, where, orderCol, orderDir, argIdx, argIdx+1)
	args = append(args, f.PerPage, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	out := []domain.Notification{}
	for rows.Next() {
		var n domain.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Event, &n.Title, &n.Body, &n.Href, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan notification: %w", err)
		}
		out = append(out, n)
	}
	return out, total, nil
}

// Stats returns the total/unread/read counts in a single round-trip.
func (r *NotificationRepo) Stats(ctx context.Context, userID uuid.UUID) (domain.NotificationStats, error) {
	var s domain.NotificationStats
	err := r.pool.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE read_at IS NULL),
		       count(*) FILTER (WHERE read_at IS NOT NULL)
		FROM notifications WHERE user_id = $1`, userID).
		Scan(&s.Total, &s.Unread, &s.Read)
	if err != nil {
		return domain.NotificationStats{}, fmt.Errorf("notification stats: %w", err)
	}
	return s, nil
}

// GetByID returns one notification, scoped to its owner (a user can never read
// another user's notification).
func (r *NotificationRepo) GetByID(ctx context.Context, userID, id uuid.UUID) (*domain.Notification, error) {
	var n domain.Notification
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, event, title, body, href, read_at, created_at
		FROM notifications WHERE id = $1 AND user_id = $2`, id, userID).
		Scan(&n.ID, &n.UserID, &n.Event, &n.Title, &n.Body, &n.Href, &n.ReadAt, &n.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotificationNotFound
		}
		return nil, fmt.Errorf("get notification: %w", err)
	}
	return &n, nil
}

// MarkReadMany marks several notifications read in one statement, scoped to the
// owner so a forged id can never touch another user's row.
func (r *NotificationRepo) MarkReadMany(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET read_at = now() WHERE user_id = $1 AND id = ANY($2) AND read_at IS NULL`,
		userID, ids)
	if err != nil {
		return fmt.Errorf("mark notifications read: %w", err)
	}
	return nil
}

// Delete removes one notification owned by the user.
func (r *NotificationRepo) Delete(ctx context.Context, userID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM notifications WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete notification: %w", err)
	}
	return nil
}

// DeleteMany removes several notifications in one statement, scoped to the owner.
func (r *NotificationRepo) DeleteMany(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM notifications WHERE user_id = $1 AND id = ANY($2)`, userID, ids)
	if err != nil {
		return fmt.Errorf("delete notifications: %w", err)
	}
	return nil
}

func (r *NotificationRepo) MarkRead(ctx context.Context, userID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET read_at = now() WHERE id = $1 AND user_id = $2 AND read_at IS NULL`, id, userID)
	return err
}

func (r *NotificationRepo) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET read_at = now() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}

func (r *NotificationRepo) Clear(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM notifications WHERE user_id = $1`, userID)
	return err
}

var _ domain.NotificationRepository = (*NotificationRepo)(nil)
