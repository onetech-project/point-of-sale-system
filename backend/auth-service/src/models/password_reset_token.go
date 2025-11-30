package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type PasswordResetToken struct {
	ID        uuid.UUID    `json:"id" db:"id"`
	UserID    uuid.UUID    `json:"user_id" db:"user_id"`
	TenantID  uuid.UUID    `json:"tenant_id" db:"tenant_id"`
	Token     string       `json:"token" db:"token"`
	ExpiresAt time.Time    `json:"expires_at" db:"expires_at"`
	UsedAt    sql.NullTime `json:"used_at,omitempty" db:"used_at"`
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
}
