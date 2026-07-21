package crypto

import (
	"fmt"
	"time"
)

// Service provides the public API for all crypto operations
type Service struct {
	engine *Engine
	store  *KeyStore
}

// NewService creates a crypto service with all algorithms registered
func NewService(store *KeyStore) *Service {
	e := NewEngine()
	e.Register(&aesGCMCrypter{})
	e.Register(&aesCBCCrypter{})
	e.Register(&chacha20Crypter{})
	e.Register(&rsaCrypter{})
	e.Register(&x25519Crypter{})
	e.Register(&argon2Crypter{})
	e.Register(&ed25519Crypter{})
	e.Register(&ecdsaCrypter{})

	return &Service{engine: e, store: store}
}

// CreateKey generates a new encryption key for a user
func (s *Service) CreateKey(userID string, algo Algorithm, purpose KeyPurpose) (*EncryptionKey, error) {
	c, err := s.engine.GetCrypter(algo)
	if err != nil {
		return nil, err
	}
	key, err := c.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	key.ID = GenKeyID()
	key.UserID = userID
	key.Algorithm = algo
	key.Purpose = purpose
	key.CreatedAt = time.Now().Unix()

	if err := s.store.Save(key); err != nil {
		return nil, fmt.Errorf("save key: %w", err)
	}
	return key, nil
}

// Encrypt encrypts plaintext using the specified key
func (s *Service) Encrypt(req *EncryptRequest, userID string) (*EncryptResult, error) {
	key, err := s.store.Get(req.KeyID)
	if err != nil {
		return nil, fmt.Errorf("key not found: %w", err)
	}
	if key.UserID != userID {
		return nil, fmt.Errorf("key does not belong to user")
	}

	c, err := s.engine.GetCrypter(key.Algorithm)
	if err != nil {
		return nil, err
	}

	var aad []byte
	if req.AAD != "" {
		aad = []byte(req.AAD)
	}

	ciphertext, nonce, tag, err := c.Encrypt(key, []byte(req.Plaintext), aad)
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}

	return &EncryptResult{
		Ciphertext: encodeBase64(ciphertext),
		Nonce:      encodeBase64(nonce),
		Tag:        encodeBase64(tag),
		Algorithm:  string(key.Algorithm),
		KeyID:      key.ID,
	}, nil
}

// Decrypt decrypts ciphertext using the specified key
func (s *Service) Decrypt(req *DecryptRequest, userID string) (*DecryptResult, error) {
	key, err := s.store.Get(req.KeyID)
	if err != nil {
		return nil, fmt.Errorf("key not found: %w", err)
	}
	if key.UserID != userID {
		return nil, fmt.Errorf("key does not belong to user")
	}

	c, err := s.engine.GetCrypter(key.Algorithm)
	if err != nil {
		return nil, err
	}

	ciphertext, err := decodeBase64(req.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("ciphertext decode: %w", err)
	}
	nonce, err := decodeBase64(req.Nonce)
	if err != nil {
		return nil, fmt.Errorf("nonce decode: %w", err)
	}
	var tag []byte
	if req.Tag != "" {
		tag, err = decodeBase64(req.Tag)
		if err != nil {
			return nil, fmt.Errorf("tag decode: %w", err)
		}
	}
	var aad []byte
	if req.AAD != "" {
		aad = []byte(req.AAD)
	}

	plaintext, err := c.Decrypt(key, ciphertext, nonce, aad, tag)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return &DecryptResult{Plaintext: string(plaintext)}, nil
}

// Sign signs data using an asymmetric key (ed25519 / ecdsa-p256).
func (s *Service) Sign(userID, keyID, data string) (*SignResult, error) {
	key, err := s.store.Get(keyID)
	if err != nil {
		return nil, fmt.Errorf("key not found: %w", err)
	}
	if key.UserID != userID {
		return nil, fmt.Errorf("key does not belong to user")
	}
	switch key.Algorithm {
	case AlgoEd25519:
		c := &ed25519Crypter{}
		sig, err := c.Sign(key, []byte(data))
		if err != nil {
			return nil, err
		}
		return &SignResult{KeyID: key.ID, Algorithm: string(key.Algorithm), Signature: encodeBase64(sig)}, nil
	case AlgoECDSAP256:
		c := &ecdsaCrypter{}
		sig, err := c.Sign(key, []byte(data))
		if err != nil {
			return nil, err
		}
		return &SignResult{KeyID: key.ID, Algorithm: string(key.Algorithm), Signature: encodeBase64(sig)}, nil
	default:
		return nil, fmt.Errorf("algorithm %s does not support signing", key.Algorithm)
	}
}

// Verify verifies a signature using an asymmetric key.
func (s *Service) Verify(userID, keyID, data, signature string) (*VerifyResult, error) {
	key, err := s.store.Get(keyID)
	if err != nil {
		return nil, fmt.Errorf("key not found: %w", err)
	}
	if key.UserID != userID {
		return nil, fmt.Errorf("key does not belong to user")
	}
	sig, err := decodeBase64(signature)
	if err != nil {
		return nil, fmt.Errorf("signature decode: %w", err)
	}
	var valid bool
	switch key.Algorithm {
	case AlgoEd25519:
		c := &ed25519Crypter{}
		valid, err = c.Verify(key, []byte(data), sig)
	case AlgoECDSAP256:
		c := &ecdsaCrypter{}
		valid, err = c.Verify(key, []byte(data), sig)
	default:
		return nil, fmt.Errorf("algorithm %s does not support verification", key.Algorithm)
	}
	if err != nil {
		return nil, err
	}
	return &VerifyResult{Valid: valid}, nil
}

// Hash computes a digest using a supported hash algorithm.
func (s *Service) Hash(algo, data, encoding string) (*HashResult, error) {
	raw, err := decodeInput(data, encoding)
	if err != nil {
		return nil, err
	}
	digest, err := ComputeHash(Algorithm(algo), raw)
	if err != nil {
		return nil, err
	}
	return &HashResult{Algorithm: algo, Digest: digest}, nil
}

// ListAlgorithms returns metadata about all supported crypto primitives.
func (s *Service) ListAlgorithms() []AlgoInfo {
	return []AlgoInfo{
		{Name: string(AlgoAES256GCM), Kind: "symmetric", AEAD: true, Notes: "Fast, authenticated. Recommended default."},
		{Name: string(AlgoChaCha20Poly), Kind: "symmetric", AEAD: true, Notes: "Mobile/ARM-friendly, no AES hardware needed."},
		{Name: string(AlgoAES256CBC), Kind: "symmetric", AEAD: false, Notes: "Legacy compatibility; uses Encrypt-then-MAC (HMAC-SHA256)."},
		{Name: string(AlgoX25519AESGCM), Kind: "hybrid", AEAD: true, Notes: "ECDH key exchange + AES-GCM. Good for transport."},
		{Name: string(AlgoRSAOAEP4096), Kind: "asymmetric", AEAD: false, Notes: "Encrypt to a public key; large ciphertext."},
		{Name: string(AlgoArgon2ID), Kind: "password-hash", AEAD: false, Notes: "OWASP-recommended password hashing (memory-hard)."},
		{Name: string(AlgoEd25519), Kind: "signature", AEAD: false, Notes: "Fast signatures, 32-byte keys."},
		{Name: string(AlgoECDSAP256), Kind: "signature", AEAD: false, Notes: "NIST P-256 ECDSA signatures."},
		{Name: string(AlgoSHA256), Kind: "hash", AEAD: false, Notes: "SHA-2 family digest."},
		{Name: string(AlgoSHA512), Kind: "hash", AEAD: false, Notes: "SHA-2 family, 512-bit."},
		{Name: string(AlgoBLAKE2b), Kind: "hash", AEAD: false, Notes: "Fast, modern, keyless optional."},
		{Name: string(AlgoSHA3), Kind: "hash", AEAD: false, Notes: "Keccak/SHA-3 family."},
	}
}

// AlgoInfo describes a supported algorithm for discovery endpoints.
type AlgoInfo struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"` // symmetric | asymmetric | hybrid | signature | hash | password-hash
	AEAD  bool   `json:"aead"` // authenticated encryption?
	Notes string `json:"notes"`
}

// GetKey returns a single key (without private key material)
func (s *Service) GetKey(id string) (*EncryptionKey, error) {
	return s.store.Get(id)
}

// ListKeys returns all keys for a user
func (s *Service) ListKeys(userID string) ([]*EncryptionKey, error) {
	return s.store.List(userID)
}

// DeleteKey removes a key
func (s *Service) DeleteKey(id string) error {
	return s.store.Delete(id)
}
