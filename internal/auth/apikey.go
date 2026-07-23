package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const keyPrefixLen = 8

// APIKey represents a stored API key
type APIKey struct {
	Prefix      string   `json:"prefix"`
	Hash        string   `json:"-"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	OwnerID     string   `json:"owner_id,omitempty"`  // user who created the key (blank = system)
	DBAccess    []string `json:"db_access,omitempty"` // connection IDs this key may access
	CreatedAt   string   `json:"created_at"`
	LastUsedAt  string   `json:"last_used_at,omitempty"`
	ExpiresAt   string   `json:"expires_at,omitempty"`
}

// APIKeyService manages API key generation and validation
type APIKeyService struct {
	store KeyStore
}

// KeyStore is the interface for persisting API keys
type KeyStore interface {
	SaveKey(ctx context.Context, key APIKey) error
	GetKey(ctx context.Context, prefix string) (*APIKey, error)
	ListKeys(ctx context.Context) ([]APIKey, error)
	DeleteKey(ctx context.Context, prefix string) error
}

// NewAPIKeyService creates a new API key service
func NewAPIKeyService(store KeyStore) *APIKeyService {
	return &APIKeyService{store: store}
}

// Generate creates a new API key (prefix + raw) and stores the hash
func (s *APIKeyService) Generate(ctx context.Context, name string, permissions []string, ownerID string, dbAccess []string) (string, *APIKey, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("apikey: *** %w", err)
	}

	rawKey := hex.EncodeToString(raw)
	prefix := rawKey[:keyPrefixLen]
	hash := hashKey(rawKey)

	key := APIKey{
		Prefix:      prefix,
		Hash:        hash,
		Name:        name,
		Permissions: permissions,
		OwnerID:     ownerID,
		DBAccess:    dbAccess,
	}

	if err := s.store.SaveKey(ctx, key); err != nil {
		return "", nil, fmt.Errorf("apikey: save: %w", err)
	}

	return rawKey, &key, nil
}

// Validate checks if a raw key is valid and returns the stored key
func (s *APIKeyService) Validate(ctx context.Context, rawKey string) (*APIKey, error) {
	if len(rawKey) < keyPrefixLen {
		return nil, fmt.Errorf("apikey: invalid format")
	}

	prefix := rawKey[:keyPrefixLen]
	stored, err := s.store.GetKey(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("apikey: not found")
	}

	if stored.Hash != hashKey(rawKey) {
		return nil, fmt.Errorf("apikey: invalid key")
	}

	return stored, nil
}

// List returns all stored API keys (without hashes)
func (s *APIKeyService) List(ctx context.Context) ([]APIKey, error) {
	return s.store.ListKeys(ctx)
}

// Revoke deletes an API key by prefix
func (s *APIKeyService) Revoke(ctx context.Context, prefix string) error {
	return s.store.DeleteKey(ctx, prefix)
}

// FormatKey returns a human-readable masked key
func FormatKey(rawKey string) string {
	if len(rawKey) < 12 {
		return rawKey
	}
	return rawKey[:keyPrefixLen] + strings.Repeat("*", len(rawKey)-12) + rawKey[len(rawKey)-4:]
}

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}
