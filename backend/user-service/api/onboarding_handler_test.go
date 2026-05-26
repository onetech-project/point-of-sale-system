package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pos/user-service/src/models"
	"github.com/pos/user-service/src/services"
)

type fakeOnboardingService struct {
	progress *models.OnboardingProgress
	err      error
	seen     services.OnboardingIdentity
}

func (f *fakeOnboardingService) GetProgress(c echo.Context, identity services.OnboardingIdentity, tourKey string, tourVersion int) (*models.OnboardingProgress, error) {
	f.seen = identity
	if f.err != nil {
		return nil, f.err
	}
	return f.progress, nil
}

func (f *fakeOnboardingService) Complete(c echo.Context, identity services.OnboardingIdentity, tourKey string, tourVersion int) (*models.OnboardingProgress, error) {
	f.seen = identity
	if f.err != nil {
		return nil, f.err
	}
	return f.progress, nil
}

func TestOnboardingHandlerMissingAuth(t *testing.T) {
	e := echo.New()
	handler := NewOnboardingHandlerWithService(&fakeOnboardingService{err: services.ErrOnboardingMissingAuth})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/onboarding/progress?tour_key=owner-onboarding&tour_version=1", nil)
	rec := httptest.NewRecorder()

	if err := handler.GetProgress(e.NewContext(req, rec)); err != nil {
		t.Fatalf("GetProgress returned error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestOnboardingHandlerProgressAndComplete(t *testing.T) {
	completedAt := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		method string
		body   string
		call   func(*OnboardingHandler, echo.Context) error
	}{
		{
			name:   "progress",
			method: http.MethodGet,
			call:   (*OnboardingHandler).GetProgress,
		},
		{
			name:   "complete",
			method: http.MethodPost,
			body:   `{"tour_key":"owner-onboarding","tour_version":1}`,
			call:   (*OnboardingHandler).Complete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			fake := &fakeOnboardingService{progress: &models.OnboardingProgress{Completed: true, CompletedAt: &completedAt}}
			handler := NewOnboardingHandlerWithService(fake)
			target := "/api/v1/users/onboarding/progress?tour_key=owner-onboarding&tour_version=1"
			if tt.method == http.MethodPost {
				target = "/api/v1/users/onboarding/complete"
			}
			req := httptest.NewRequest(tt.method, target, strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			req.Header.Set("X-Tenant-ID", "tenant-1")
			req.Header.Set("X-User-ID", "user-1")
			req.Header.Set("X-User-Role", "owner")
			rec := httptest.NewRecorder()

			if err := tt.call(handler, e.NewContext(req, rec)); err != nil {
				t.Fatalf("%s returned error: %v", tt.name, err)
			}
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
			}
			if fake.seen.TenantID != "tenant-1" || fake.seen.UserID != "user-1" || fake.seen.Role != "owner" {
				t.Fatalf("identity = %+v, want gateway headers", fake.seen)
			}

			var body map[string]interface{}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("response is not JSON: %v", err)
			}
			if body["completed"] != true {
				t.Fatalf("completed = %v, want true", body["completed"])
			}
		})
	}
}

func TestOnboardingHandlerInvalidTourMismatch(t *testing.T) {
	e := echo.New()
	handler := NewOnboardingHandlerWithService(&fakeOnboardingService{err: services.ErrOnboardingInvalidTour})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/onboarding/complete", strings.NewReader(`{"tour_key":"owner-onboarding","tour_version":1}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-User-ID", "user-1")
	req.Header.Set("X-User-Role", "cashier")
	rec := httptest.NewRecorder()

	if err := handler.Complete(e.NewContext(req, rec)); err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
