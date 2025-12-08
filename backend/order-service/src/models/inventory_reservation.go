package models

import (
	"time"
)

// ReservationStatus represents the state of an inventory reservation
type ReservationStatus string

const (
	ReservationStatusActive    ReservationStatus = "active"
	ReservationStatusExpired   ReservationStatus = "expired"
	ReservationStatusConverted ReservationStatus = "converted"
	ReservationStatusReleased  ReservationStatus = "released"
)

// InventoryReservation represents a temporary hold on product inventory
type InventoryReservation struct {
	ID         string            `json:"id"`
	OrderID    string            `json:"order_id"`
	ProductID  string            `json:"product_id"`
	Quantity   int               `json:"quantity"`
	Status     ReservationStatus `json:"status"`
	CreatedAt  time.Time         `json:"created_at"`
	ExpiresAt  time.Time         `json:"expires_at"`
	ReleasedAt *time.Time        `json:"released_at,omitempty"`
}

// IsExpired checks if the reservation has expired
func (ir *InventoryReservation) IsExpired() bool {
	return time.Now().After(ir.ExpiresAt) && ir.Status == ReservationStatusActive
}

// Scan implements sql.Scanner for ReservationStatus
func (s *ReservationStatus) Scan(value interface{}) error {
	if value == nil {
		*s = ReservationStatusActive
		return nil
	}
	*s = ReservationStatus(value.(string))
	return nil
}
