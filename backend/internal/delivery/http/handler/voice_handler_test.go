package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
)

// fakeVoiceRepo is an in-memory VoiceTranscriptionRepository for handler tests.
type fakeVoiceRepo struct {
	created []domain.VoiceTranscription
	// failCreate forces Create to error (to exercise the 500 path).
	failCreate bool
}

func (f *fakeVoiceRepo) Create(_ context.Context, t *domain.VoiceTranscription) error {
	if f.failCreate {
		return context.DeadlineExceeded
	}
	t.ID = uuid.New()
	t.CreatedAt = time.Now()
	f.created = append(f.created, *t)
	return nil
}
func (f *fakeVoiceRepo) ListByUser(context.Context, uuid.UUID, int, int) ([]domain.VoiceTranscription, int, error) {
	return nil, 0, nil
}
func (f *fakeVoiceRepo) CountSince(context.Context, uuid.UUID, time.Time) (int64, error) { return 0, nil }
func (f *fakeVoiceRepo) DeleteOne(context.Context, uuid.UUID, uuid.UUID) error           { return nil }
func (f *fakeVoiceRepo) ClearByUser(context.Context, uuid.UUID) error                    { return nil }
func (f *fakeVoiceRepo) DeleteOlderThan(context.Context, time.Time) (int64, error)       { return 0, nil }
func (f *fakeVoiceRepo) IncrementDailyUsage(context.Context, uuid.UUID, time.Time) (int64, error) {
	return 1, nil
}
func (f *fakeVoiceRepo) DailyUsage(context.Context, uuid.UUID, time.Time) (int64, error) { return 0, nil }
func (f *fakeVoiceRepo) PruneDailyUsageBefore(context.Context, time.Time) (int64, error) { return 0, nil }
func (f *fakeVoiceRepo) ConsumeDailyUsage(context.Context, uuid.UUID, time.Time, int64) (bool, int64, error) {
	return true, 1, nil
}

var _ domain.VoiceTranscriptionRepository = (*fakeVoiceRepo)(nil)

// voiceRecordRouter wires POST /me/voice-history with a user_id in context.
func voiceRecordRouter(repo domain.VoiceTranscriptionRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewVoiceHandler(repo, nil, nil)
	r.POST("/me/voice-history", func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		h.Record(c)
	})
	return r
}

func postVoiceHistory(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := voiceRecordRouter(&fakeVoiceRepo{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/me/voice-history", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestVoiceRecord_RejectsEmptyTranscript(t *testing.T) {
	w := postVoiceHistory(t, `{"transcript":"   ","mode":"browser"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestVoiceRecord_RejectsTooLongTranscript(t *testing.T) {
	body := `{"transcript":"` + strings.Repeat("a", maxTranscriptLen+1) + `"}`
	w := postVoiceHistory(t, body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestVoiceRecord_RejectsTooLongLanguage(t *testing.T) {
	// Must be a 400, not a 500 from the varchar(16) column.
	body := `{"transcript":"halo","language":"` + strings.Repeat("x", maxLanguageLen+1) + `"}`
	w := postVoiceHistory(t, body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestVoiceRecord_ClampsNegativeDuration(t *testing.T) {
	repo := &fakeVoiceRepo{}
	r := voiceRecordRouter(repo)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/me/voice-history",
		strings.NewReader(`{"transcript":"halo","duration_ms":-5000}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	if len(repo.created) != 1 {
		t.Fatalf("created %d rows, want 1", len(repo.created))
	}
	if repo.created[0].DurationMS != 0 {
		t.Fatalf("duration = %d, want 0 (clamped)", repo.created[0].DurationMS)
	}
}

func TestVoiceRecord_DefaultsLanguage(t *testing.T) {
	repo := &fakeVoiceRepo{}
	r := voiceRecordRouter(repo)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/me/voice-history",
		strings.NewReader(`{"transcript":"halo"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	if repo.created[0].Language != "id" {
		t.Fatalf("language = %q, want default 'id'", repo.created[0].Language)
	}
	if repo.created[0].Mode != domain.VoiceModeBrowser {
		t.Fatalf("mode = %q, want browser", repo.created[0].Mode)
	}
}
