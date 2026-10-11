package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestNormalizeCombo(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "default"},
		{"   ", "default"},
		{"custom", "custom"},
		{"  backup  ", "backup"},
	}

	for _, tc := range cases {
		got := normalizeCombo(tc.in)
		if got != tc.want {
			t.Errorf("normalizeCombo(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestUniqueIDs(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()

	input := []uuid.UUID{id1, id2, id1, id2, id1}
	got := uniqueIDs(input)

	if len(got) != 2 {
		t.Fatalf("expected 2 unique IDs, got %d", len(got))
	}
	if (got[0] != id1 && got[0] != id2) || (got[1] != id1 && got[1] != id2) || got[0] == got[1] {
		t.Errorf("unexpected unique IDs: %v", got)
	}
}

func TestPutRoutes_ValidationPayload(t *testing.T) {
	h := &Handler{svc: &Service{}}
	r := setupTestRouter()
	r.PUT("/routes", h.PutRoutes)

	validID := uuid.New().String()

	cases := []struct {
		name       string
		body       []routeDTO
		wantStatus int
		wantError  string
	}{
		{
			name: "Invalid use_case",
			body: []routeDTO{
				{UseCase: "telepathy", EndpointID: validID, Model: "m1", Priority: 1},
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "validation",
		},
		{
			name: "Invalid endpoint_id format",
			body: []routeDTO{
				{UseCase: "chat", EndpointID: "not-a-uuid", Model: "m1", Priority: 1},
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_payload",
		},
		{
			name: "Negative timeout",
			body: []routeDTO{
				{UseCase: "chat", EndpointID: validID, Model: "m1", Priority: 1, TimeoutMS: -1},
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "validation",
		},
		{
			name: "Negative max_retries",
			body: []routeDTO{
				{UseCase: "chat", EndpointID: validID, Model: "m1", Priority: 1, MaxRetries: -5},
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "validation",
		},
		{
			name: "Duplicate route same combo",
			body: []routeDTO{
				{UseCase: "chat", EndpointID: validID, Model: "m1", Combo: "default", Priority: 1},
				{UseCase: "chat", EndpointID: validID, Model: "m1", Combo: "default", Priority: 2},
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "duplicate_route",
		},
		{
			name: "Duplicate route empty combo normalized to default",
			body: []routeDTO{
				{UseCase: "chat", EndpointID: validID, Model: "m1", Combo: "", Priority: 1},
				{UseCase: "chat", EndpointID: validID, Model: "m1", Combo: "default", Priority: 2},
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "duplicate_route",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPut, "/routes", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", w.Code, tc.wantStatus, w.Body.String())
			}

			var resp struct {
				Error string `json:"error"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &resp)
			if resp.Error != tc.wantError {
				t.Errorf("error = %q, want %q", resp.Error, tc.wantError)
			}
		})
	}
}
