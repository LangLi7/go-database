package auth

import (
	"fmt"
	"time"
)

type Claims struct {
	UserID        string   `json:"user_id"`
	Username      string   `json:"username"`
	Role          string   `json:"role"`
	ExtraPerm     []string `json:"extra_perm,omitempty"`
	ExtraDBAccess []string `json:"extra_db_access,omitempty"`
	ExpiresAt     int64    `json:"exp"`
}

type JWTService struct {
	impl     *TokenService
	duration time.Duration
}

func NewJWTService(secret string, durationMinutes int) (*JWTService, error) {
	ts, err := NewTokenService(secret, durationMinutes)
	if err != nil {
		return nil, fmt.Errorf("auth: initialize token service: %w", err)
	}
	return &JWTService{impl: ts, duration: time.Duration(durationMinutes) * time.Minute}, nil
}

func (s *JWTService) MasterKey() []byte {
	return s.impl.MasterKey()
}

func (s *JWTService) GenerateToken(userID, username, role string, extraPerm, extraDBAccess []string) (string, error) {
	return s.impl.GenerateToken(userID, username, role, extraPerm, extraDBAccess)
}

func (s *JWTService) ValidateToken(tokenStr string) (*Claims, error) {
	tc, err := s.impl.ValidateToken(tokenStr)
	if err != nil {
		return nil, err
	}
	return &Claims{
		UserID:        tc.UserID,
		Username:      tc.Username,
		Role:          tc.Role,
		ExtraPerm:     tc.ExtraPerm,
		ExtraDBAccess: tc.ExtraDBAccess,
		ExpiresAt:     tc.ExpiresAt,
	}, nil
}
