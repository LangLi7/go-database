package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims holds the JWT token payload
type Claims struct {
	UserID    string   `json:"user_id"`
	Username  string   `json:"username"`
	Role      string   `json:"role"`
	ExtraPerm []string `json:"extra_perm,omitempty"`
	jwt.RegisteredClaims
}

// JWTService handles token creation and validation
type JWTService struct {
	secret     []byte
	duration   time.Duration
}

// NewJWTService creates a new JWT service
func NewJWTService(secret string, durationMinutes int) *JWTService {
	return &JWTService{
		secret:   []byte(secret),
		duration: time.Duration(durationMinutes) * time.Minute,
	}
}

// GenerateToken creates a signed JWT for a user
func (s *JWTService) GenerateToken(userID, username, role string, extraPerm []string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		Username:  username,
		Role:      role,
		ExtraPerm: extraPerm,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "go-database",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("jwt: sign: %w", err)
	}
	return signed, nil
}

// ValidateToken parses and validates a JWT token
func (s *JWTService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwt: unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("jwt: validate: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("jwt: invalid token")
	}
	return claims, nil
}
