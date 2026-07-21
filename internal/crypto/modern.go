package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"math/big"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
)

// ─── Argon2id (modern password hashing, OWASP-recommended) ─────────

// Argon2idParams are sensible, conservative defaults (RFC/OWASP-aligned).
const (
	argonTime    = 1
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
)

type argon2Crypter struct{}

func (a *argon2Crypter) Algorithm() Algorithm { return AlgoArgon2ID }

// GenerateKey produces a fresh salt; the "key" here is the salt used for hashing.
func (a *argon2Crypter) GenerateKey() (*EncryptionKey, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("argon2 salt: %w", err)
	}
	return &EncryptionKey{PrivKey: salt}, nil
}

func (a *argon2Crypter) Encrypt(key *EncryptionKey, plaintext, aad []byte) (ciphertext, nonce, tag []byte, err error) {
	// "plaintext" is the password; "nonce" carries the salt (in PrivKey or aad).
	salt := key.PrivKey
	if len(salt) == 0 {
		salt = aad
	}
	if len(salt) < argonSaltLen {
		return nil, nil, nil, fmt.Errorf("argon2: salt too short")
	}
	hash := argon2.IDKey(plaintext, salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	// Return the raw hash as ciphertext; salt goes into nonce for round-trip.
	return hash, salt, nil, nil
}

func (a *argon2Crypter) Decrypt(key *EncryptionKey, ciphertext, nonce, aad, tag []byte) ([]byte, error) {
	// Argon2 is one-way; "decrypt" re-computes the hash for verification.
	salt := nonce
	if len(salt) == 0 {
		salt = key.PrivKey
	}
	if len(salt) < argonSaltLen {
		return nil, fmt.Errorf("argon2: salt too short")
	}
	recomputed := argon2.IDKey(ciphertext, salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return recomputed, nil
}

// ─── ed25519 signatures ────────────────────────────────────────────

type ed25519Crypter struct{}

func (e *ed25519Crypter) Algorithm() Algorithm { return AlgoEd25519 }

func (e *ed25519Crypter) GenerateKey() (*EncryptionKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("ed25519: %w", err)
	}
	return &EncryptionKey{PublicKey: pub, PrivKey: priv}, nil
}

func (e *ed25519Crypter) Encrypt(key *EncryptionKey, plaintext, aad []byte) (ciphertext, nonce, tag []byte, err error) {
	// Sign, not encrypt. Return signature in ciphertext.
	sig := ed25519.Sign(key.PrivKey, plaintext)
	return sig, nil, nil, nil
}

func (e *ed25519Crypter) Decrypt(key *EncryptionKey, ciphertext, nonce, aad, tag []byte) ([]byte, error) {
	// Verify; "plaintext" (ciphertext here) is the original message to verify against.
	// We store the message in `ciphertext` slot on sign, so verify needs both.
	// For API symmetry, verification is done via Verify() below, not Decrypt.
	return ciphertext, nil
}

// Sign signs data with an ed25519 private key.
func (e *ed25519Crypter) Sign(key *EncryptionKey, data []byte) ([]byte, error) {
	return ed25519.Sign(key.PrivKey, data), nil
}

// Verify checks an ed25519 signature.
func (e *ed25519Crypter) Verify(key *EncryptionKey, data, sig []byte) (bool, error) {
	return ed25519.Verify(key.PublicKey, data, sig), nil
}

// ─── ECDSA P-256 signatures ────────────────────────────────────────

type ecdsaCrypter struct{}

func (e *ecdsaCrypter) Algorithm() Algorithm { return AlgoECDSAP256 }

func (e *ecdsaCrypter) GenerateKey() (*EncryptionKey, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("ecdsa: %w", err)
	}
	pubBytes, err := x509MarshalPKIX(&priv.PublicKey)
	if err != nil {
		return nil, err
	}
	privBytes, err := x509MarshalECPrivate(priv)
	if err != nil {
		return nil, err
	}
	return &EncryptionKey{PublicKey: pubBytes, PrivKey: privBytes}, nil
}

func (e *ecdsaCrypter) Encrypt(key *EncryptionKey, plaintext, aad []byte) (ciphertext, nonce, tag []byte, err error) {
	// ECDSA signs; return ASN.1 DER signature in ciphertext.
	priv, err := x509ParseECPrivate(key.PrivKey)
	if err != nil {
		return nil, nil, nil, err
	}
	digest := sha256.Sum256(plaintext)
	r, s, err := ecdsa.Sign(rand.Reader, priv, digest[:])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("ecdsa sign: %w", err)
	}
	sig := append(r.Bytes(), s.Bytes()...) // simplified (r||s), 64 bytes
	return sig, nil, nil, nil
}

func (e *ecdsaCrypter) Decrypt(key *EncryptionKey, ciphertext, nonce, aad, tag []byte) ([]byte, error) {
	return ciphertext, nil
}

// Sign signs data with ECDSA-P256 private key (returns r||s, 64 bytes).
func (e *ecdsaCrypter) Sign(key *EncryptionKey, data []byte) ([]byte, error) {
	priv, err := x509ParseECPrivate(key.PrivKey)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, priv, digest[:])
	if err != nil {
		return nil, fmt.Errorf("ecdsa sign: %w", err)
	}
	return append(r.Bytes(), s.Bytes()...), nil
}

// Verify checks an ECDSA-P256 signature (r||s, 64 bytes).
func (e *ecdsaCrypter) Verify(key *EncryptionKey, data, sig []byte) (bool, error) {
	pub, err := x509ParsePKIXPublic(key.PublicKey)
	if err != nil {
		return false, err
	}
	ecPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return false, fmt.Errorf("not an ECDSA public key")
	}
	if len(sig) != 64 {
		return false, fmt.Errorf("ecdsa: signature must be 64 bytes (r||s)")
	}
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:])
	digest := sha256.Sum256(data)
	return ecdsa.Verify(ecPub, digest[:], r, s), nil
}

// ─── Hash / digest functions (server-computed) ────────────────────

// ComputeHash returns the hex digest of data using the named algorithm.
func ComputeHash(algo Algorithm, data []byte) (string, error) {
	var h hash.Hash
	switch algo {
	case AlgoSHA256:
		h = sha256.New()
	case AlgoSHA512:
		h = sha512.New()
	case AlgoBLAKE2b:
		b, err := blake2b.New256(nil)
		if err != nil {
			return "", err
		}
		h = b
	case AlgoSHA3:
		h = sha3.New256()
	default:
		return "", fmt.Errorf("unsupported hash algorithm: %s", algo)
	}
	if _, err := io.Copy(h, bytesReader(data)); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Helper: decode input data per encoding ("utf8"|"base64"|"hex").
func decodeInput(data, encoding string) ([]byte, error) {
	switch encoding {
	case "", "utf8", "text":
		return []byte(data), nil
	case "base64":
		return base64.StdEncoding.DecodeString(data)
	case "hex":
		return hex.DecodeString(data)
	default:
		return nil, fmt.Errorf("unsupported input encoding: %s", encoding)
	}
}
