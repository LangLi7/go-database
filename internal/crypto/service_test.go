package crypto

import (
	"path/filepath"
	"testing"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	storePath := filepath.Join(t.TempDir(), "keys.db")
	store, err := NewKeyStore(storePath, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("new keystore: %v", err)
	}
	return NewService(store)
}

func TestArgon2IDRoundTrip(t *testing.T) {
	svc := newTestService(t)
	key, err := svc.CreateKey("u1", AlgoArgon2ID, PurposeDataEncryption)
	if err != nil {
		t.Fatalf("create argon2 key: %v", err)
	}
	plain := "mySecretPassword"
	res, err := svc.Encrypt(&EncryptRequest{KeyID: key.ID, Plaintext: plain}, "u1")
	if err != nil {
		t.Fatalf("argon2 hash: %v", err)
	}
	// Argon2 is one-way: hashing the same password twice (same key) yields the
	// same digest (salt is fixed in the key's PrivKey).
	res2, err := svc.Encrypt(&EncryptRequest{KeyID: key.ID, Plaintext: plain}, "u1")
	if err != nil {
		t.Fatalf("argon2 hash 2: %v", err)
	}
	if res.Ciphertext != res2.Ciphertext {
		t.Error("argon2: same password+key produced different hashes (salt must be stable)")
	}
	// Different password must produce different hash
	res3, _ := svc.Encrypt(&EncryptRequest{KeyID: key.ID, Plaintext: "otherPassword"}, "u1")
	if res.Ciphertext == res3.Ciphertext {
		t.Error("argon2: identical hashes for different passwords (should differ via salt)")
	}
}

func TestEd25519SignVerify(t *testing.T) {
	svc := newTestService(t)
	key, err := svc.CreateKey("u1", AlgoEd25519, PurposeSigning)
	if err != nil {
		t.Fatalf("create ed25519 key: %v", err)
	}
	data := "transaction #42"
	sig, err := svc.Sign("u1", key.ID, data)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if sig.Signature == "" {
		t.Fatal("empty signature")
	}
	ok, err := svc.Verify("u1", key.ID, data, sig.Signature)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok.Valid {
		t.Error("valid signature reported invalid")
	}
	bad, _ := svc.Verify("u1", key.ID, "transaction #43", sig.Signature)
	if bad.Valid {
		t.Error("tampered data verified as valid")
	}
}

func TestECDSAP256SignVerify(t *testing.T) {
	svc := newTestService(t)
	key, err := svc.CreateKey("u1", AlgoECDSAP256, PurposeSigning)
	if err != nil {
		t.Fatalf("create ecdsa key: %v", err)
	}
	data := "payload"
	sig, err := svc.Sign("u1", key.ID, data)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	ok, err := svc.Verify("u1", key.ID, data, sig.Signature)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok.Valid {
		t.Error("ecdsa signature invalid")
	}
}

func TestHashFunctions(t *testing.T) {
	svc := newTestService(t)
	for _, algo := range []string{"sha256", "sha512", "blake2b", "sha3-256"} {
		r, err := svc.Hash(algo, "hallo", "utf8")
		if err != nil {
			t.Fatalf("hash %s: %v", algo, err)
		}
		if r.Digest == "" {
			t.Errorf("hash %s: empty digest", algo)
		}
		t.Logf("%s(\"hallo\") = %s", algo, r.Digest)
	}
}

func TestListAlgorithmsCount(t *testing.T) {
	svc := newTestService(t)
	algos := svc.ListAlgorithms()
	if len(algos) < 12 {
		t.Errorf("expected >=12 algorithms, got %d", len(algos))
	}
}
