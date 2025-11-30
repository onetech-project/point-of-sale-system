package models

import (
	"time"
)

type InvitationStatus string

const (
	InvitationPending  InvitationStatus = "pending"
	InvitationAccepted InvitationStatus = "accepted"
	InvitationExpired  InvitationStatus = "expired"
	InvitationRevoked  InvitationStatus = "revoked"
)

type Invitation struct {
	ID         string           `json:"id" db:"id"`
	TenantID   string           `json:"tenantId" db:"tenant_id"`
	Email      string           `json:"email" db:"email"`
	Role       string           `json:"role" db:"role"`
	Token      string           `json:"token" db:"token"`
	Status     InvitationStatus `json:"status" db:"status"`
	InvitedBy  string           `json:"invitedBy" db:"invited_by"`
	ExpiresAt  time.Time        `json:"expiresAt" db:"expires_at"`
	AcceptedAt *time.Time       `json:"acceptedAt,omitempty" db:"accepted_at"`
	CreatedAt  time.Time        `json:"createdAt" db:"created_at"`
	UpdatedAt  time.Time        `json:"updatedAt" db:"updated_at"`
}

type InvitationRequest struct {
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"required,oneof=admin manager cashier"`
}

type InvitationAcceptRequest struct {
	FirstName string `json:"firstName" validate:"required,min=2,max=50"`
	LastName  string `json:"lastName" validate:"required,min=2,max=50"`
	Password  string `json:"password" validate:"required,min=8"`
}

type InvitationResponse struct {
	ID        string           `json:"id"`
	Email     string           `json:"email"`
	Role      string           `json:"role"`
	Status    InvitationStatus `json:"status"`
	ExpiresAt time.Time        `json:"expiresAt"`
	InvitedBy string           `json:"invitedBy"`
	CreatedAt time.Time        `json:"createdAt"`
}
