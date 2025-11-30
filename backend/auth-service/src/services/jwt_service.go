package services

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type JWTService struct {
	secret     []byte
	expiration time.Duration
}

type JWTClaims struct {
	SessionID string `json:"sessionId"`
	UserID    string `json:"userId"`
	TenantID  string `json:"tenantId"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

func NewJWTService(secret string, expirationMinutes int) *JWTService {
	return &JWTService{
		secret:     []byte(secret),
		expiration: time.Duration(expirationMinutes) * time.Minute,
	}
}

// Generate creates a new JWT token
func (s *JWTService) Generate(sessionID, userID, tenantID, email, role string) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		SessionID: sessionID,
		UserID:    userID,
		TenantID:  tenantID,
		Email:     email,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "pos-auth-service",
			Subject:   userID,
			ID:        sessionID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token: %w", err)
	}

	return tokenString, nil
}

// Validate validates and parses a JWT token
func (s *JWTService) Validate(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid JWT token")
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, fmt.Errorf("failed to parse JWT claims")
	}

	return claims, nil
}

// ExtractClaims extracts claims without full validation (for debugging)
func (s *JWTService) ExtractClaims(tokenString string) (*JWTClaims, error) {
	token, _ := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return s.secret, nil
	})

	if claims, ok := token.Claims.(*JWTClaims); ok {
		return claims, nil
	}

	return nil, fmt.Errorf("failed to extract claims")
}

// RefreshToken creates a new token with extended expiration
func (s *JWTService) RefreshToken(oldTokenString string) (string, error) {
	claims, err := s.Validate(oldTokenString)
	if err != nil {
		return "", fmt.Errorf("cannot refresh invalid token: %w", err)
	}

	// Generate new token with same claims but new expiration
	return s.Generate(claims.SessionID, claims.UserID, claims.TenantID, claims.Email, claims.Role)
}
