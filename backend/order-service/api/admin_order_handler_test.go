package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/point-of-sale-system/order-service/src/models"
)

type fakeOrderService struct {
	order *models.GuestOrder
	items []models.OrderItem
	notes []*models.OrderNote
	err   error
}

func (f *fakeOrderService) ListOrdersByTenant(ctx context.Context, tenantID string, status *models.OrderStatus, limit, offset int) ([]*models.GuestOrder, error) {
	return nil, nil
}

func (f *fakeOrderService) GetOrderByID(ctx context.Context, orderID string) (*models.GuestOrder, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.order, nil
}

func (f *fakeOrderService) GetOrderItems(ctx context.Context, orderID string) ([]models.OrderItem, error) {
	return f.items, nil
}

func (f *fakeOrderService) GetOrderNotes(ctx context.Context, orderID string) ([]*models.OrderNote, error) {
	return f.notes, nil
}

func (f *fakeOrderService) UpdateOrderStatus(ctx context.Context, orderID string, newStatus models.OrderStatus) error {
	return nil
}

func (f *fakeOrderService) AddOrderNote(ctx context.Context, orderID, note, userName string) error {
	return nil
}

func TestAdminOrderHandlerGetOrderUsesTenantHeaderAndReturnsDetails(t *testing.T) {
	now := time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)
	createdBy := "Manager"
	handler := NewAdminOrderHandler(&fakeOrderService{
		order: &models.GuestOrder{
			ID:               "order-1",
			OrderReference:   "GO-0001",
			TenantID:         "tenant-1",
			Status:           models.OrderStatusPending,
			SubtotalAmount:   30000,
			TotalAmount:      30000,
			CustomerName:     "Customer",
			CustomerPhone:    "+628123456789",
			DeliveryType:     models.DeliveryTypePickup,
			CreatedAt:        now,
			DataConsentGiven: true,
		},
		items: []models.OrderItem{
			{
				ID:          "item-1",
				OrderID:     "order-1",
				ProductID:   "product-1",
				ProductName: "Coffee",
				Quantity:    2,
				UnitPrice:   15000,
				TotalPrice:  30000,
				CreatedAt:   now,
			},
		},
		notes: []*models.OrderNote{
			{
				ID:            "note-1",
				OrderID:       "order-1",
				Note:          "Latest note",
				CreatedByName: &createdBy,
				CreatedAt:     now,
			},
		},
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/orders/order-1", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("order-1")

	if err := handler.GetOrder(c); err != nil {
		t.Fatalf("GetOrder returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response struct {
		Order      models.GuestOrder  `json:"order"`
		Items      []models.OrderItem `json:"items"`
		LatestNote *models.OrderNote  `json:"latest_note"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Order.ID != "order-1" {
		t.Fatalf("order id = %s, want order-1", response.Order.ID)
	}
	if len(response.Items) != 1 || response.Items[0].ProductName != "Coffee" {
		t.Fatalf("items = %+v, want Coffee item", response.Items)
	}
	if response.LatestNote == nil || response.LatestNote.Note != "Latest note" {
		t.Fatalf("latest_note = %+v, want latest note", response.LatestNote)
	}
}

func TestAdminOrderHandlerGetOrderRejectsCrossTenantAccess(t *testing.T) {
	handler := NewAdminOrderHandler(&fakeOrderService{
		order: &models.GuestOrder{
			ID:             "order-1",
			OrderReference: "GO-0001",
			TenantID:       "tenant-2",
			Status:         models.OrderStatusPending,
			DeliveryType:   models.DeliveryTypePickup,
			CreatedAt:      time.Now(),
		},
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/orders/order-1", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("order-1")

	if err := handler.GetOrder(c); err != nil {
		t.Fatalf("GetOrder returned error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}
