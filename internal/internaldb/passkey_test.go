package internaldb

import (
	"context"
	"path/filepath"
	"testing"

	"go-database/internal/auth"
)

func TestPasskeyRoundTrip(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	// Seed a user (required FK-style owner).
	u := auth.User{ID: "user-1", Username: "alice", PasswordHash: "x", Role: "readonly"}
	if err := store.SaveUser(ctx, u); err != nil {
		t.Fatalf("save user: %v", err)
	}

	pk := &auth.Passkey{
		ID:           "pk-1",
		UserID:       "user-1",
		Name:         "Laptop TPM",
		PublicKey:    []byte("public-key-bytes"),
		CredentialID: []byte("cred-id-bytes"),
		Attestation:  "none",
		AAGUID:       "00112233445566778899aabbccddeeff",
		SignCount:    5,
		CreatedAt:    123,
	}
	if err := store.SavePasskey(ctx, pk); err != nil {
		t.Fatalf("save passkey: %v", err)
	}

	// List by user
	list, err := store.ListPasskeys(ctx, "user-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Name != "Laptop TPM" {
		t.Fatalf("unexpected list: %+v", list)
	}

	// Lookup by credential ID (login path)
	got, err := store.GetPasskeyByCredentialID(ctx, []byte("cred-id-bytes"))
	if err != nil {
		t.Fatalf("get by cred id: %v", err)
	}
	if got.AAGUID != pk.AAGUID || got.SignCount != 5 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}

	// Update sign count
	if err := store.UpdatePasskeySignCount(ctx, "pk-1", 9); err != nil {
		t.Fatalf("update sign count: %v", err)
	}
	got, _ = store.GetPasskeyByCredentialID(ctx, []byte("cred-id-bytes"))
	if got.SignCount != 9 {
		t.Fatalf("sign count not updated: %d", got.SignCount)
	}

	// Delete
	if err := store.DeletePasskey(ctx, "pk-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, _ = store.ListPasskeys(ctx, "user-1")
	if len(list) != 0 {
		t.Fatalf("expected empty after delete, got %d", len(list))
	}
}
