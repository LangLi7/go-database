package auth

import (
	"context"
	"fmt"
	"testing"
)

func TestNewJWTService_CustomSecret(t *testing.T) {
	svc, err := NewJWTService("my-secret-key-1234567890123456", 60)
	if err != nil {
		t.Fatalf("failed to create JWT service: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewJWTService_EmptySecret(t *testing.T) {
	svc, err := NewJWTService("", 60)
	if err != nil {
		t.Fatalf("failed to create JWT service: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	svc, err := NewJWTService("test-secret-32byte-key-here!!!!", 60)
	if err != nil {
		t.Fatalf("failed to create JWT service: %v", err)
	}

	token, err := svc.GenerateToken("user1", "testuser", "admin", []string{"connections:list"}, nil)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}
	if claims.UserID != "user1" {
		t.Fatalf("expected user1, got %s", claims.UserID)
	}
	if claims.Role != "admin" {
		t.Fatalf("expected admin, got %s", claims.Role)
	}
	if len(claims.ExtraPerm) != 1 || claims.ExtraPerm[0] != "connections:list" {
		t.Fatalf("unexpected extra_perm: %v", claims.ExtraPerm)
	}
}

func TestValidateInvalidToken(t *testing.T) {
	svc, err := NewJWTService("test-secret", 60)
	if err != nil {
		t.Fatalf("failed to create JWT service: %v", err)
	}

	_, err = svc.ValidateToken("invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestGenerateAndValidateWithEmptyPerms(t *testing.T) {
	svc, err := NewJWTService("test-secret", 60)
	if err != nil {
		t.Fatalf("failed to create JWT service: %v", err)
	}

	token, err := svc.GenerateToken("u1", "un", "dev", nil, nil)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}
	if claims.UserID != "u1" || claims.Username != "un" || claims.Role != "dev" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestAPIKeyService(t *testing.T) {
	store := &memKeyStore{keys: make(map[string]APIKey)}
	svc := NewAPIKeyService(store)

	ctx := context.Background()
	key, stored, err := svc.Generate(ctx, "test-key", []string{"connections:list"})
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	if key == "" {
		t.Fatal("expected non-empty raw key")
	}
	if stored.Name != "test-key" {
		t.Fatalf("expected test-key, got %s", stored.Name)
	}

	validated, err := svc.Validate(ctx, key)
	if err != nil {
		t.Fatalf("failed to validate key: %v", err)
	}
	if validated.Prefix != stored.Prefix {
		t.Fatalf("prefix mismatch: %s != %s", validated.Prefix, stored.Prefix)
	}

	keys, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("failed to list keys: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}

	err = svc.Revoke(ctx, stored.Prefix)
	if err != nil {
		t.Fatalf("failed to revoke key: %v", err)
	}

	_, err = svc.Validate(ctx, key)
	if err == nil {
		t.Fatal("expected error for revoked key")
	}
}

// memKeyStore is an in-memory KeyStore for tests
type memKeyStore struct {
	keys map[string]APIKey
}

func (m *memKeyStore) SaveKey(ctx context.Context, key APIKey) error {
	m.keys[key.Prefix] = key
	return nil
}

func (m *memKeyStore) GetKey(ctx context.Context, prefix string) (*APIKey, error) {
	k, ok := m.keys[prefix]
	if !ok {
		return nil, errNotFound
	}
	return &k, nil
}

func (m *memKeyStore) ListKeys(ctx context.Context) ([]APIKey, error) {
	var list []APIKey
	for _, k := range m.keys {
		list = append(list, k)
	}
	return list, nil
}

func (m *memKeyStore) DeleteKey(ctx context.Context, prefix string) error {
	delete(m.keys, prefix)
	return nil
}

var errNotFound = fmt.Errorf("not found")
