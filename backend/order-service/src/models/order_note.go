package models

import "time"

// OrderNote represents a note/comment added to an order
// Used for courier tracking, admin comments, status updates, etc.
type OrderNote struct {
	ID              string    `json:"id"`
	OrderID         string    `json:"order_id"`
	Note            string    `json:"note"`
	CreatedByUserID *string   `json:"created_by_user_id,omitempty"`
	CreatedByName   *string   `json:"created_by_name,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// CreateOrderNoteRequest represents the request to create a note
type CreateOrderNoteRequest struct {
	Note            string  `json:"note" validate:"required,min=1,max=5000"`
	CreatedByUserID *string `json:"created_by_user_id,omitempty"`
	CreatedByName   *string `json:"created_by_name,omitempty"`
}
