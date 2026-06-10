package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type TokenClaims struct {
	UserID    string   `json:"uid"`
	Username  string   `json:"un"`
	Role      string   `json:"role"`
	ExtraPerm []string `json:"ep,omitempty"`
	ExpiresAt int64    `json:"exp"`
}

type TokenService struct {
	key      [32]byte
	duration time.Duration
}

func NewTokenService(secret string, durationMinutes int) (*TokenService, error) {
	var key [32]byte
	if secret != "" {
		copy(key[:], secret)
	} else {
		var err error
		key, err = loadOrGenerateKey()
		if err != nil {
			return nil, fmt.Errorf("token: key setup: %w", err)
		}
	}
	return &TokenService{
		key:      key,
		duration: time.Duration(durationMinutes) * time.Minute,
	}, nil
}

func (s *TokenService) GenerateToken(userID, username, role string, extraPerm []string) (string, error) {
	claims := TokenClaims{
		UserID:    userID,
		Username:  username,
		Role:      role,
		ExtraPerm: extraPerm,
		ExpiresAt: time.Now().Add(s.duration).Unix(),
	}
	plain, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("token: marshal: %w", err)
	}
	return encrypt(s.key[:], plain)
}

func (s *TokenService) ValidateToken(tokenStr string) (*TokenClaims, error) {
	plain, err := decrypt(s.key[:], tokenStr)
	if err != nil {
		return nil, fmt.Errorf("token: decrypt: %w", err)
	}
	var claims TokenClaims
	if err := json.Unmarshal(plain, &claims); err != nil {
		return nil, fmt.Errorf("token: unmarshal: %w", err)
	}
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("token: expired")
	}
	return &claims, nil
}

func loadOrGenerateKey() ([32]byte, error) {
	var key [32]byte
	dir, err := keyDir()
	if err != nil {
		return key, err
	}
	keyPath := filepath.Join(dir, "secret.key")
	if data, err := os.ReadFile(keyPath); err == nil && len(data) == 32 {
		copy(key[:], data)
		return key, nil
	}
	if _, err := rand.Read(key[:]); err != nil {
		return key, fmt.Errorf("keygen: %w", err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return key, fmt.Errorf("keydir: %w", err)
	}
	if err := os.WriteFile(keyPath, key[:], 0600); err != nil {
		return key, fmt.Errorf("keywrite: %w", err)
	}
	return key, nil
}

func keyDir() (string, error) {
	if dir := os.Getenv("GODB_KEY_DIR"); dir != "" {
		return dir, nil
	}
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(appData, "go-database"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home: %w", err)
	}
	return filepath.Join(home, ".config", "go-database"), nil
}

func encrypt(key []byte, plaintext []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return base64.RawURLEncoding.EncodeToString(append(nonce, ciphertext...)), nil
}

func decrypt(key []byte, tokenStr string) ([]byte, error) {
	data, err := base64.RawURLEncoding.DecodeString(tokenStr)
	if err != nil {
		return nil, fmt.Errorf("base64: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("token too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
