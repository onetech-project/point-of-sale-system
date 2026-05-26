package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/pos/api-gateway/middleware"
)

func TestUserOnboardingRoutesAllowAllTenantRoles(t *testing.T) {
	for _, role := range []middleware.Role{middleware.RoleOwner, middleware.RoleManager, middleware.RoleCashier} {
		t.Run(string(role), func(t *testing.T) {
			var sawGet, sawPost bool
			userService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Tenant-ID") != "tenant-1" || r.Header.Get("X-User-ID") != "user-1" || r.Header.Get("X-User-Role") != string(role) {
					t.Fatalf("gateway auth headers were not forwarded")
				}
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/api/v1/users/onboarding/progress":
					sawGet = true
					if r.URL.Query().Get("tour_key") != "owner-onboarding" || r.URL.Query().Get("tour_version") != "1" {
						t.Fatalf("query was not forwarded: %s", r.URL.RawQuery)
					}
				case r.Method == http.MethodPost && r.URL.Path == "/api/v1/users/onboarding/complete":
					sawPost = true
				default:
					t.Fatalf("unexpected proxied request: %s %s", r.Method, r.URL.String())
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			defer userService.Close()

			gateway := onboardingRouteGateway(userService.URL, role)
			getReq := httptest.NewRequest(http.MethodGet, "/api/v1/users/onboarding/progress?tour_key=owner-onboarding&tour_version=1", nil)
			getRec := httptest.NewRecorder()
			gateway.ServeHTTP(getRec, getReq)
			if getRec.Code != http.StatusOK {
				t.Fatalf("GET status = %d, want %d; body=%s", getRec.Code, http.StatusOK, getRec.Body.String())
			}

			postReq := httptest.NewRequest(http.MethodPost, "/api/v1/users/onboarding/complete", strings.NewReader(`{"tour_key":"owner-onboarding","tour_version":1}`))
			postReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			postRec := httptest.NewRecorder()
			gateway.ServeHTTP(postRec, postReq)
			if postRec.Code != http.StatusOK {
				t.Fatalf("POST status = %d, want %d; body=%s", postRec.Code, http.StatusOK, postRec.Body.String())
			}
			if !sawGet || !sawPost {
				t.Fatalf("sawGet=%t sawPost=%t, want both true", sawGet, sawPost)
			}
		})
	}
}

func onboardingRouteGateway(userServiceURL string, role middleware.Role) *echo.Echo {
	e := echo.New()
	protected := e.Group("")
	protected.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("role", string(role))
			c.Set("tenant_id", "tenant-1")
			c.Set("user_id", "user-1")
			return next(c)
		}
	})
	registerUserOnboardingRoutes(protected, userServiceURL)
	return e
}
