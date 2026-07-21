package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// Algorithm identifies the encryption method
type Algorithm string

const (
	AlgoAES256GCM    Algorithm = "aes-256-gcm"
	AlgoAES256CBC    Algorithm = "aes-256-cbc"
	AlgoChaCha20Poly Algorithm = "chacha20-poly1305"
	AlgoRSAOAEP4096  Algorithm = "rsa-oaep-4096"
	AlgoX25519AESGCM Algorithm = "x25519-aes-256-gcm"
	// Modern password hashing (Argon2id is the current OWASP-recommended standard)
	AlgoArgon2ID Algorithm = "argon2id"
	// Asymmetric signatures
	AlgoEd25519   Algorithm = "ed25519"
	AlgoECDSAP256 Algorithm = "ecdsa-p256"
	// Hash / digest functions (not encryption, but crypto primitives)
	AlgoSHA256  Algorithm = "sha256"
	AlgoSHA512  Algorithm = "sha512"
	AlgoBLAKE2b Algorithm = "blake2b"
	AlgoSHA3    Algorithm = "sha3-256"
)

// PurposeHash marks a key/operation as a digest (not encryption/signing).
const PurposeHash KeyPurpose = "hash"

// HashRequest asks the server to compute a digest.
type HashRequest struct {
	Algorithm string `json:"algorithm" binding:"required"`
	Data      string `json:"data" binding:"required"`
	Encoding  string `json:"encoding,omitempty"` // "utf8" (default) | "base64" | "hex" (input)
}

// HashResult holds the computed digest.
type HashResult struct {
	Algorithm string `json:"algorithm"`
	Digest    string `json:"digest"` // hex-encoded
}

// SignRequest signs data with an asymmetric key.
type SignRequest struct {
	KeyID string `json:"key_id" binding:"required"`
	Data  string `json:"data" binding:"required"`
}

// SignResult holds the signature.
type SignResult struct {
	KeyID     string `json:"key_id"`
	Algorithm string `json:"algorithm"`
	Signature string `json:"signature"` // base64
}

// VerifyRequest verifies a signature.
type VerifyRequest struct {
	KeyID     string `json:"key_id" binding:"required"`
	Data      string `json:"data" binding:"required"`
	Signature string `json:"signature" binding:"required"` // base64
}

// VerifyResult reports signature validity.
type VerifyResult struct {
	Valid bool `json:"valid"`
}

// KeyPurpose describes what a key is used for
type KeyPurpose string

const (
	PurposeColumnEncryption KeyPurpose = "column-encryption"
	PurposeDataEncryption   KeyPurpose = "data-encryption"
	PurposeKeyEncryption    KeyPurpose = "key-encryption"
	PurposeSigning          KeyPurpose = "signing"
)

// EncryptionKey holds key material for a single encryption key
type EncryptionKey struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Algorithm Algorithm  `json:"algorithm"`
	Purpose   KeyPurpose `json:"purpose"`
	PublicKey []byte     `json:"public_key,omitempty"`
	PrivKey   []byte     `json:"-"` // never serialized to JSON
	CreatedAt int64      `json:"created_at"`
	ExpiresAt int64      `json:"expires_at,omitempty"`
	Version   int        `json:"version"`
}

// EncryptRequest describes what to encrypt
type EncryptRequest struct {
	KeyID     string `json:"key_id"`
	Plaintext string `json:"plaintext"`
	Encoding  string `json:"encoding,omitempty"` // "base64" or "hex" (default base64)
	AAD       string `json:"aad,omitempty"`      // additional authenticated data (AEAD only)
}

// EncryptResult holds the encryption output
type EncryptResult struct {
	Ciphertext string `json:"ciphertext"`
	Nonce      string `json:"nonce"`
	Algorithm  string `json:"algorithm"`
	KeyID      string `json:"key_id"`
	Tag        string `json:"tag,omitempty"`
}

// DecryptRequest describes what to decrypt
type DecryptRequest struct {
	KeyID      string `json:"key_id"`
	Ciphertext string `json:"ciphertext"`
	Nonce      string `json:"nonce"`
	Algorithm  string `json:"algorithm"`
	Encoding   string `json:"encoding,omitempty"`
	AAD        string `json:"aad,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

// DecryptResult holds decrypted data
type DecryptResult struct {
	Plaintext string `json:"plaintext"`
}

// Crypter implements a specific encryption algorithm
type Crypter interface {
	Algorithm() Algorithm
	GenerateKey() (*EncryptionKey, error)
	Encrypt(key *EncryptionKey, plaintext, aad []byte) (ciphertext, nonce, tag []byte, err error)
	Decrypt(key *EncryptionKey, ciphertext, nonce, aad, tag []byte) ([]byte, error)
}

// Engine orchestrates all crypto operations
type Engine struct {
	keys  map[string]*EncryptionKey
	algos map[Algorithm]Crypter
}

// NewEngine creates a crypto engine
func NewEngine() *Engine {
	return &Engine{
		keys:  make(map[string]*EncryptionKey),
		algos: make(map[Algorithm]Crypter),
	}
}

// Register adds a crypter implementation
func (e *Engine) Register(c Crypter) {
	e.algos[c.Algorithm()] = c
}

// GetCrypter returns the implementation for an algorithm
func (e *Engine) GetCrypter(algo Algorithm) (Crypter, error) {
	c, ok := e.algos[algo]
	if !ok {
		return nil, fmt.Errorf("unsupported algorithm: %s", algo)
	}
	return c, nil
}

// randomBytes generates cryptographically secure random bytes
func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("random: %w", err)
	}
	return b, nil
}

// padPKCS7 applies PKCS7 padding
func padPKCS7(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	p := make([]byte, padding)
	for i := range p {
		p[i] = byte(padding)
	}
	return append(data, p...)
}

// unpadPKCS7 removes PKCS7 padding
func unpadPKCS7(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	padding := int(data[len(data)-1])
	if padding > len(data) || padding > 256 {
		return nil, fmt.Errorf("invalid padding")
	}
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return nil, fmt.Errorf("invalid padding byte")
		}
	}
	return data[:len(data)-padding], nil
}

// encodeBase64 encodes bytes to base64 string
func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// decodeBase64 decodes base64 string to bytes
func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
