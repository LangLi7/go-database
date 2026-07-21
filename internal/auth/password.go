package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// Argon2id parameters — match crypto/modern.go for consistency.
const (
	argonTime    = 1
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
)

// HashPassword hashes a password with Argon2id (modern, OWASP-recommended).
// Existing bcrypt hashes remain valid via CheckPassword's dual-verify.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := randRead(salt); err != nil {
		return "", fmt.Errorf("auth: salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	// Format: argon2id$time$memory$threads$saltB64$hashB64  (salt kept for verify)
	return fmt.Sprintf("argon2id$%d$%d$%d$%s$%s",
		argonTime, argonMemory, argonThreads,
		b64(salt), b64(hash)), nil
}

// CheckPassword verifies against Argon2id or legacy bcrypt (dual-verify).
func CheckPassword(password, stored string) error {
	if strings.HasPrefix(stored, "argon2id$") {
		return checkArgon2(password, stored)
	}
	// Legacy bcrypt ($2a$/$2b$): still accepted so existing accounts keep working.
	return bcrypt.CompareHashAndPassword([]byte(stored), []byte(password))
}

func checkArgon2(password, stored string) error {
	parts := strings.Split(stored, "$")
	if len(parts) != 6 {
		return fmt.Errorf("auth: malformed argon2 hash")
	}
	salt, err := unb64(parts[4])
	if err != nil {
		return fmt.Errorf("auth: bad salt: %w", err)
	}
	expected, err := unb64(parts[5])
	if err != nil {
		return fmt.Errorf("auth: bad hash: %w", err)
	}
	t, m, th := argonTime, argonMemory, argonThreads
	computed := argon2.IDKey([]byte(password), salt, uint32(t), uint32(m), uint8(th), argonKeyLen)
	if subtle.ConstantTimeEq(int32(len(computed)), int32(len(expected))) != 1 {
		return fmt.Errorf("auth: invalid password")
	}
	if subtle.ConstantTimeCompare(computed, expected) != 1 {
		return fmt.Errorf("auth: invalid password")
	}
	return nil
}

// DefaultAdminHash returns the hash of the default "admin" password.
func DefaultAdminHash() (string, error) {
	return HashPassword("admin")
}

// IsDefaultPassword returns true if the hash matches the default "admin" password.
func IsDefaultPassword(hash string) bool {
	return CheckPassword("admin", hash) == nil
}

func randRead(b []byte) (int, error) { return rand.Read(b) }
func b64(b []byte) string            { return base64.RawStdEncoding.EncodeToString(b) }
func unb64(s string) ([]byte, error) { return base64.RawStdEncoding.DecodeString(s) }
