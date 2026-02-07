package models

import (
	"time"
)

// Session represents an authenticated user session
type Session struct {
	ID           string     `json:"id"`
	SessionID    string     `json:"sessionId"`
	TenantID     string     `json:"tenantId"`
	UserID       string     `json:"userId"`
	IPAddress    string     `json:"ipAddress,omitempty"`
	UserAgent    string     `json:"userAgent,omitempty"`
	ExpiresAt    time.Time  `json:"expiresAt"`
	TerminatedAt *time.Time `json:"terminatedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

// SessionData represents the session data stored in Redis
type SessionData struct {
	UserID    string `json:"userId"`
	TenantID  string `json:"tenantId"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	CreatedAt int64  `json:"createdAt"`
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	User    UserInfo `json:"user"`
	Message string   `json:"message"`
}

// UserInfo represents user information returned in login response
type UserInfo struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	TenantID  string `json:"tenantId"`
	Role      string `json:"role"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Locale    string `json:"locale"`
}

// SessionResponse represents session validation response
type SessionResponse struct {
	Valid    bool      `json:"valid"`
	User     *UserInfo `json:"user,omitempty"`
	TenantID string    `json:"tenantId,omitempty"`
}
