package models

import (
	"time"
)

type User struct {
	ID           string     `json:"id" db:"id"`
	TenantID     string     `json:"tenant_id" db:"tenant_id"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"`
	Role         string     `json:"role" db:"role"`
	Status       string     `json:"status" db:"status"`
	FirstName    *string    `json:"first_name,omitempty" db:"first_name"`
	LastName     *string    `json:"last_name,omitempty" db:"last_name"`
	Locale       string     `json:"locale" db:"locale"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

type UserRole string

const (
	RoleOwner   UserRole = "owner"
	RoleManager UserRole = "manager"
	RoleCashier UserRole = "cashier"
)

type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInvited   UserStatus = "invited"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusDeleted   UserStatus = "deleted"
)

type CreateUserRequest struct {
	TenantID  string  `json:"tenant_id" validate:"required,uuid"`
	Email     string  `json:"email" validate:"required,email"`
	Password  string  `json:"password" validate:"required,min=8"`
	Role      string  `json:"role" validate:"required,oneof=owner manager cashier"`
	FirstName *string `json:"first_name,omitempty" validate:"omitempty,max=50"`
	LastName  *string `json:"last_name,omitempty" validate:"omitempty,max=50"`
	Locale    string  `json:"locale,omitempty" validate:"omitempty,oneof=en id"`
}

type UserResponse struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	FirstName   *string    `json:"first_name,omitempty"`
	LastName    *string    `json:"last_name,omitempty"`
	Locale      string     `json:"locale"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:          u.ID,
		TenantID:    u.TenantID,
		Email:       u.Email,
		Role:        u.Role,
		Status:      u.Status,
		FirstName:   u.FirstName,
		LastName:    u.LastName,
		Locale:      u.Locale,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
	}
}
