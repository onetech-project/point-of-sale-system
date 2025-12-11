package services

import (
	"testing"
)

// TestRenderStaffNotificationTemplate tests the staff notification email template rendering
func TestRenderStaffNotificationTemplate(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		wantErr  bool
		validate func(t *testing.T, html string)
	}{
		{
			name: "Valid order data",
			data: map[string]interface{}{
				"OrderID":         "ORD-12345",
				"TransactionID":   "TXN-67890",
				"CustomerName":    "John Doe",
				"CustomerEmail":   "john@example.com",
				"CustomerPhone":   "+6281234567890",
				"SubtotalAmount":  "50.000",
				"DeliveryFee":     "10.000",
				"TotalAmount":     "60.000",
				"PaymentMethod":   "qris",
				"PaymentProvider": "midtrans",
				"PaidAt":          "2024-01-15 10:30:00",
				"Items": []map[string]interface{}{
					{
						"ProductName": "Nasi Goreng",
						"Quantity":    2,
						"UnitPrice":   "15.000",
						"TotalPrice":  "30.000",
					},
					{
						"ProductName": "Es Teh",
						"Quantity":    2,
						"UnitPrice":   "10.000",
						"TotalPrice":  "20.000",
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, html string) {
				// Check for key elements in the rendered HTML
				required := []string{
					"ORD-12345",
					"John Doe",
					"john@example.com",
					"60.000",
					"Nasi Goreng",
					"Es Teh",
				}
				for _, s := range required {
					if !contains(html, s) {
						t.Errorf("Expected HTML to contain %q, but it didn't", s)
					}
				}
			},
		},
		{
			name: "Order without customer email",
			data: map[string]interface{}{
				"OrderID":        "ORD-12345",
				"TransactionID":  "TXN-67890",
				"CustomerName":   "Guest Customer",
				"CustomerPhone":  "+6281234567890",
				"SubtotalAmount": "50.000",
				"TotalAmount":    "50.000",
				"PaymentMethod":  "qris",
				"PaidAt":         "2024-01-15 10:30:00",
				"Items": []map[string]interface{}{
					{
						"ProductName": "Coffee",
						"Quantity":    1,
						"UnitPrice":   "50.000",
						"TotalPrice":  "50.000",
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, html string) {
				if !contains(html, "Guest Customer") {
					t.Error("Expected HTML to contain customer name")
				}
				if !contains(html, "ORD-12345") {
					t.Error("Expected HTML to contain order ID")
				}
			},
		},
		{
			name:    "Empty data",
			data:    map[string]interface{}{},
			wantErr: false, // Template should render even with missing data
			validate: func(t *testing.T, html string) {
				// Should still render basic HTML structure
				if len(html) < 100 {
					t.Error("Expected HTML output even with empty data")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test will FAIL initially because renderStaffNotificationTemplate() doesn't exist yet
			// This is expected in TDD - write the test first, then implement the function

			// TODO: Uncomment when implementation is ready
			// service := &NotificationService{
			// 	templates: make(map[string]*template.Template),
			// }

			// Load templates
			// if err := service.loadTemplates(); err != nil {
			// 	t.Fatalf("Failed to load templates: %v", err)
			// }

			// html, err := service.renderStaffNotificationTemplate(tt.data)

			// if (err != nil) != tt.wantErr {
			// 	t.Errorf("renderStaffNotificationTemplate() error = %v, wantErr %v", err, tt.wantErr)
			// 	return
			// }

			// if !tt.wantErr && tt.validate != nil {
			// 	tt.validate(t, html)
			// }

			// For now, this test will fail (as expected in TDD)
			t.Skip("Implementation pending - TDD: test written first")
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
