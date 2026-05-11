package services

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/pos/notification-service/src/models"
	"github.com/pos/notification-service/src/utils"
)

func TestHandleBillingNotificationEvents(t *testing.T) {
	tests := []struct {
		name        string
		event       models.NotificationEvent
		template    string
		wantSubject string
		wantBody    string
	}{
		{
			name: "trial started sends dedicated template",
			event: models.NotificationEvent{
				EventType: "subscription.trial_started",
				TenantID:  "tenant-1",
				Data: map[string]interface{}{
					"email":         "owner@example.com",
					"tenant_name":   "Test Cafe",
					"trial_ends_at": "2026-05-18T00:00:00Z",
				},
			},
			template:    "trial_started",
			wantSubject: "Your 7-day free trial has started!",
			wantBody:    "Test Cafe",
		},
		{
			name: "trial ending accepts canonical tenant_name",
			event: models.NotificationEvent{
				EventType: "subscription.trial_ending",
				TenantID:  "tenant-1",
				Data: map[string]interface{}{
					"email":          "owner@example.com",
					"tenant_name":    "Test Cafe",
					"days_remaining": float64(3),
					"trial_ends_at":  "2026-05-18T00:00:00Z",
				},
			},
			template:    "trial_ending",
			wantSubject: "Your Posku trial expires in 3 day(s)",
			wantBody:    "Test Cafe",
		},
		{
			name: "invoice paid sends payment confirmation",
			event: models.NotificationEvent{
				EventType: "invoice.paid",
				TenantID:  "tenant-1",
				Data: map[string]interface{}{
					"email":            "owner@example.com",
					"tenant_name":      "Test Cafe",
					"invoice_number":   "INV-202605-000001",
					"amount_idr":       float64(299000),
					"billing_interval": "monthly",
					"period_start":     "2026-05-11T00:00:00Z",
					"period_end":       "2026-06-11T00:00:00Z",
				},
			},
			template:    "invoice_paid",
			wantSubject: "Payment Confirmed - Invoice INV-202605-000001",
			wantBody:    "INV-202605-000001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeNotificationRepository{}
			email := &fakeEmailProvider{}
			svc := newTestNotificationService(t, repo, email, tt.template)

			payload, err := json.Marshal(tt.event)
			if err != nil {
				t.Fatalf("failed to marshal event: %v", err)
			}
			if err := svc.HandleEvent(context.Background(), payload); err != nil {
				t.Fatalf("HandleEvent returned error: %v", err)
			}

			if len(repo.created) != 1 {
				t.Fatalf("created notifications = %d, want 1", len(repo.created))
			}
			if len(email.sent) != 1 {
				t.Fatalf("sent emails = %d, want 1", len(email.sent))
			}
			if repo.created[0].Subject != tt.wantSubject {
				t.Fatalf("subject = %q, want %q", repo.created[0].Subject, tt.wantSubject)
			}
			if !strings.Contains(repo.created[0].Body, tt.wantBody) {
				t.Fatalf("body does not contain %q", tt.wantBody)
			}
			if repo.updatedStatus != models.NotificationStatusSent {
				t.Fatalf("updated status = %q, want %q", repo.updatedStatus, models.NotificationStatusSent)
			}
		})
	}
}

func TestHandleBillingNotificationEventsMissingEmail(t *testing.T) {
	tests := []struct {
		name    string
		event   models.NotificationEvent
		wantErr bool
	}{
		{
			name: "trial started skips missing email",
			event: models.NotificationEvent{
				EventType: "subscription.trial_started",
				TenantID:  "tenant-1",
				Data:      map[string]interface{}{},
			},
			wantErr: false,
		},
		{
			name: "trial ending rejects missing email",
			event: models.NotificationEvent{
				EventType: "subscription.trial_ending",
				TenantID:  "tenant-1",
				Data: map[string]interface{}{
					"days_remaining": float64(1),
				},
			},
			wantErr: true,
		},
		{
			name: "invoice paid rejects missing email",
			event: models.NotificationEvent{
				EventType: "invoice.paid",
				TenantID:  "tenant-1",
				Data:      map[string]interface{}{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeNotificationRepository{}
			email := &fakeEmailProvider{}
			svc := newTestNotificationService(t, repo, email, "trial_started", "trial_ending", "invoice_paid")

			payload, err := json.Marshal(tt.event)
			if err != nil {
				t.Fatalf("failed to marshal event: %v", err)
			}
			err = svc.HandleEvent(context.Background(), payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("HandleEvent error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func newTestNotificationService(t *testing.T, repo *fakeNotificationRepository, email *fakeEmailProvider, templateNames ...string) *NotificationService {
	t.Helper()

	svc := &NotificationService{
		repo:          repo,
		emailProvider: email,
		templates:     make(map[string]*template.Template),
		frontendURL:   "http://localhost:3000",
	}

	for _, name := range templateNames {
		loadTestTemplate(t, svc, name)
	}
	return svc
}

func loadTestTemplate(t *testing.T, svc *NotificationService, name string) {
	t.Helper()

	path := "../../templates/" + name + ".html"
	tmpl, err := template.New(name + ".html").Funcs(utils.GetTemplateFuncMap()).ParseFiles(path)
	if err != nil {
		t.Fatalf("failed to parse template %s: %v", name, err)
	}
	svc.templates[name] = tmpl
}

type sentEmail struct {
	to      string
	subject string
	body    string
	isHTML  bool
}

type fakeEmailProvider struct {
	sent []sentEmail
}

func (f *fakeEmailProvider) Send(to, subject, body string, isHTML bool) error {
	f.sent = append(f.sent, sentEmail{to: to, subject: subject, body: body, isHTML: isHTML})
	return nil
}

type fakeNotificationRepository struct {
	created       []*models.Notification
	updatedStatus models.NotificationStatus
}

func (f *fakeNotificationRepository) Create(_ context.Context, notification *models.Notification) error {
	if notification.ID == "" {
		notification.ID = "notification-1"
	}
	f.created = append(f.created, notification)
	return nil
}

func (f *fakeNotificationRepository) HasSentOrderNotification(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (f *fakeNotificationRepository) UpdateStatus(_ context.Context, _ string, status models.NotificationStatus, _ *time.Time, _ *time.Time, _ *string) error {
	f.updatedStatus = status
	return nil
}

func (f *fakeNotificationRepository) GetNotificationHistory(_ map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, nil
}

func (f *fakeNotificationRepository) CountNotifications(_ map[string]interface{}) (int, error) {
	return 0, nil
}

func (f *fakeNotificationRepository) GetByID(_ string) (*models.Notification, error) {
	return nil, nil
}

func (f *fakeNotificationRepository) Update(_ *models.Notification) error {
	return nil
}
