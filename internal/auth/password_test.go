package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordArgon2AndBcryptFallback(t *testing.T) {
	// New hashes are Argon2id and verify.
	h, err := HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	if h[:8] != "argon2id" {
		t.Fatalf("expected argon2id hash, got %q", h[:20])
	}
	if err := CheckPassword("secret123", h); err != nil {
		t.Errorf("argon2 verify failed: %v", err)
	}
	if err := CheckPassword("wrong", h); err == nil {
		t.Error("wrong password should fail")
	}

	// Legacy bcrypt hashes still verify (existing accounts keep working).
	legacy, _ := bcrypt.GenerateFromPassword([]byte("oldpass"), bcrypt.MinCost)
	if err := CheckPassword("oldpass", string(legacy)); err != nil {
		t.Errorf("bcrypt fallback failed: %v", err)
	}
	if err := CheckPassword("nope", string(legacy)); err == nil {
		t.Error("bcrypt wrong password should fail")
	}
}
