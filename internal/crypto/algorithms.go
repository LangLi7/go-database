package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
	"io"
)

// ─── AES-256-GCM ───────────────────────────────────────────────────

type aesGCMCrypter struct{}

func (a *aesGCMCrypter) Algorithm() Algorithm { return AlgoAES256GCM }

func (a *aesGCMCrypter) GenerateKey() (*EncryptionKey, error) {
	key, err := randomBytes(32)
	if err != nil {
		return nil, err
	}
	return &EncryptionKey{PrivKey: key}, nil
}

func (a *aesGCMCrypter) Encrypt(key *EncryptionKey, plaintext, aad []byte) (ciphertext, nonce, tag []byte, err error) {
	block, err := aes.NewCipher(key.PrivKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gcm: %w", err)
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, nil, fmt.Errorf("nonce: %w", err)
	}
	out := gcm.Seal(nil, nonce, plaintext, aad)
	// GCM appends tag to ciphertext (last 16 bytes)
	if len(out) < 16 {
		return nil, nil, nil, fmt.Errorf("gcm: output too short")
	}
	ciphertext = out[:len(out)-16]
	tag = out[len(out)-16:]
	return
}

func (a *aesGCMCrypter) Decrypt(key *EncryptionKey, ciphertext, nonce, aad, tag []byte) ([]byte, error) {
	block, err := aes.NewCipher(key.PrivKey)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	combined := append(ciphertext, tag...)
	return gcm.Open(nil, nonce, combined, aad)
}

// ─── AES-256-CBC + HMAC-SHA256 (Encrypt-then-MAC) ───────────────────

type aesCBCCrypter struct{}

func (a *aesCBCCrypter) Algorithm() Algorithm { return AlgoAES256CBC }

func (a *aesCBCCrypter) GenerateKey() (*EncryptionKey, error) {
	key := make([]byte, 64) // 32 for AES + 32 for HMAC
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("keygen: %w", err)
	}
	return &EncryptionKey{PrivKey: key}, nil
}

func (a *aesCBCCrypter) Encrypt(key *EncryptionKey, plaintext, aad []byte) (ciphertext, nonce, tag []byte, err error) {
	if len(key.PrivKey) < 64 {
		return nil, nil, nil, fmt.Errorf("key too short: need 64 bytes")
	}
	aesKey := key.PrivKey[:32]
	macKey := key.PrivKey[32:64]

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("aes: %w", err)
	}
	nonce = make([]byte, aes.BlockSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, nil, fmt.Errorf("iv: %w", err)
	}
	padded := padPKCS7(plaintext, aes.BlockSize)
	ciphertext = make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, nonce)
	mode.CryptBlocks(ciphertext, padded)

	// HMAC-SHA256(nonce || ciphertext || aad)
	h := hmac.New(sha256.New, macKey)
	h.Write(nonce)
	h.Write(ciphertext)
	if aad != nil {
		h.Write(aad)
	}
	tag = h.Sum(nil)
	return
}

func (a *aesCBCCrypter) Decrypt(key *EncryptionKey, ciphertext, nonce, aad, tag []byte) ([]byte, error) {
	if len(key.PrivKey) < 64 {
		return nil, fmt.Errorf("key too short: need 64 bytes")
	}
	aesKey := key.PrivKey[:32]
	macKey := key.PrivKey[32:64]

	// Verify HMAC first
	h := hmac.New(sha256.New, macKey)
	h.Write(nonce)
	h.Write(ciphertext)
	if aad != nil {
		h.Write(aad)
	}
	expected := h.Sum(nil)
	if !hmac.Equal(tag, expected) {
		return nil, fmt.Errorf("hmac mismatch: data tampered")
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext not multiple of block size")
	}
	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, nonce)
	mode.CryptBlocks(plaintext, ciphertext)
	return unpadPKCS7(plaintext)
}

// ─── ChaCha20-Poly1305 ─────────────────────────────────────────────

type chacha20Crypter struct{}

func (c *chacha20Crypter) Algorithm() Algorithm { return AlgoChaCha20Poly }

func (c *chacha20Crypter) GenerateKey() (*EncryptionKey, error) {
	key := make([]byte, 32) // XChaCha20-Poly1305 uses 256-bit key
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("keygen: %w", err)
	}
	return &EncryptionKey{PrivKey: key}, nil
}

func (c *chacha20Crypter) Encrypt(key *EncryptionKey, plaintext, aad []byte) (ciphertext, nonce, tag []byte, err error) {
	aead, err := chacha20poly1305.NewX(key.PrivKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("chacha20poly1305: %w", err)
	}
	nonce = make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, nil, fmt.Errorf("nonce: %w", err)
	}
	out := aead.Seal(nil, nonce, plaintext, aad)
	if len(out) < 16 {
		return nil, nil, nil, fmt.Errorf("output too short")
	}
	ciphertext = out[:len(out)-16]
	tag = out[len(out)-16:]
	return
}

func (c *chacha20Crypter) Decrypt(key *EncryptionKey, ciphertext, nonce, aad, tag []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key.PrivKey)
	if err != nil {
		return nil, fmt.Errorf("chacha20poly1305: %w", err)
	}
	combined := append(ciphertext, tag...)
	return aead.Open(nil, nonce, combined, aad)
}

// ─── RSA-OAEP-4096 ─────────────────────────────────────────────────

type rsaCrypter struct{}

func (r *rsaCrypter) Algorithm() Algorithm { return AlgoRSAOAEP4096 }

func (r *rsaCrypter) GenerateKey() (*EncryptionKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("rsa keygen: %w", err)
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("marshal pub: %w", err)
	}
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	return &EncryptionKey{PublicKey: pubBytes, PrivKey: privBytes}, nil
}

func (r *rsaCrypter) Encrypt(key *EncryptionKey, plaintext, aad []byte) (ciphertext, nonce, tag []byte, err error) {
	pub, err := x509.ParsePKIXPublicKey(key.PublicKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse pub: %w", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, nil, nil, fmt.Errorf("not an RSA public key")
	}
	label := aad
	if label == nil {
		label = []byte("")
	}
	ciphertext, err = rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPub, plaintext, label)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("rsa encrypt: %w", err)
	}
	// RSA doesn't use nonce/tag in the AEAD sense — return empty
	return ciphertext, nil, nil, nil
}

func (r *rsaCrypter) Decrypt(key *EncryptionKey, ciphertext, nonce, aad, tag []byte) ([]byte, error) {
	priv, err := x509.ParsePKCS1PrivateKey(key.PrivKey)
	if err != nil {
		return nil, fmt.Errorf("parse priv: %w", err)
	}
	label := aad
	if label == nil {
		label = []byte("")
	}
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, ciphertext, label)
	if err != nil {
		return nil, fmt.Errorf("rsa decrypt: %w", err)
	}
	return plaintext, nil
}

// ─── X25519 + HKDF + AES-256-GCM hybrid ────────────────────────────

type x25519Crypter struct{}

func (x *x25519Crypter) Algorithm() Algorithm { return AlgoX25519AESGCM }

func (x *x25519Crypter) GenerateKey() (*EncryptionKey, error) {
	priv := make([]byte, 32)
	if _, err := rand.Read(priv); err != nil {
		return nil, fmt.Errorf("keygen: %w", err)
	}
	pub, err := curve25519.X25519(priv, curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("x25519 pub: %w", err)
	}
	return &EncryptionKey{PublicKey: pub, PrivKey: priv}, nil
}

func (x *x25519Crypter) Encrypt(key *EncryptionKey, plaintext, aad []byte) (ciphertext, nonce, tag []byte, err error) {
	// Generate ephemeral key pair
	ephPriv := make([]byte, 32)
	if _, err := rand.Read(ephPriv); err != nil {
		return nil, nil, nil, fmt.Errorf("eph key: %w", err)
	}
	ephPub, err := curve25519.X25519(ephPriv, curve25519.Basepoint)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("eph pub: %w", err)
	}
	// ECDH
	shared, err := curve25519.X25519(ephPriv, key.PublicKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("ecdh: %w", err)
	}
	// Derive AES key via HKDF
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, nil, nil, fmt.Errorf("salt: %w", err)
	}
	kdf := hkdf.New(sha256.New, shared, salt, []byte("x25519-aes-256-gcm"))
	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(kdf, aesKey); err != nil {
		return nil, nil, nil, fmt.Errorf("kdf: %w", err)
	}
	// Encrypt with AES-256-GCM using derived key
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gcm: %w", err)
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, nil, fmt.Errorf("nonce: %w", err)
	}
	out := gcm.Seal(nil, nonce, plaintext, aad)
	if len(out) < 16 {
		return nil, nil, nil, fmt.Errorf("output too short")
	}
	ciphertext = make([]byte, 0, len(ephPub)+len(salt)+len(out)-16)
	ciphertext = append(ciphertext, ephPub...)
	ciphertext = append(ciphertext, salt...)
	ciphertext = append(ciphertext, out[:len(out)-16]...)
	tag = out[len(out)-16:]
	return
}

func (x *x25519Crypter) Decrypt(key *EncryptionKey, ciphertext, nonce, aad, tag []byte) ([]byte, error) {
	if len(ciphertext) < 48 { // 32 (ephPub) + 16 (salt) + minimum ciphertext
		return nil, fmt.Errorf("ciphertext too short")
	}
	ephPub := ciphertext[:32]
	salt := ciphertext[32:48]
	actualCT := ciphertext[48:]
	// ECDH with recipient's private key
	shared, err := curve25519.X25519(key.PrivKey, ephPub)
	if err != nil {
		return nil, fmt.Errorf("ecdh: %w", err)
	}
	// Derive AES key via HKDF
	kdf := hkdf.New(sha256.New, shared, salt, []byte("x25519-aes-256-gcm"))
	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(kdf, aesKey); err != nil {
		return nil, fmt.Errorf("kdf: %w", err)
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	combined := append(actualCT, tag...)
	return gcm.Open(nil, nonce, combined, aad)
}
