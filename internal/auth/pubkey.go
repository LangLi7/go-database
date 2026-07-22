package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// PubKeyAuth implements SSH-style challenge-response login: no password, the
// user proves possession of the private key by signing a server-issued nonce.
// The public key is stored on the user; the private key stays on the client.

// KeyPair is an Ed25519 keypair. PrivateKey is the raw seed — keep it secret.
type KeyPair struct {
	PublicKey  string `json:"public_key"`  // ssh-ed25519 AAAA... (base64 of raw pubkey)
	PrivateKey string `json:"private_key"` // base64 of ed25519 seed
}

// GenerateKeyPair creates a new Ed25519 keypair in ssh-key format.
func GenerateKeyPair() (KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return KeyPair{}, err
	}
	return KeyPair{
		PublicKey:  "ssh-ed25519 " + base64.StdEncoding.EncodeToString(pub),
		PrivateKey: base64.StdEncoding.EncodeToString(priv.Seed()),
	}, nil
}

// Challenge is a single-use nonce the client must sign.
type Challenge struct {
	Username  string `json:"username"`
	Nonce     string `json:"nonce"`
	ExpiresAt int64  `json:"expires_at"`
}

// NewChallenge issues a time-limited nonce for the given user.
func NewChallenge(username string) Challenge {
	return Challenge{
		Username:  username,
		Nonce:     fmt.Sprintf("%x", randUint64()),
		ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
	}
}

// SignChallenge signs the nonce with the private key (ssh-ed25519 seed form).
func SignChallenge(privKeyB64, nonce string) (string, error) {
	seed, err := base64.StdEncoding.DecodeString(privKeyB64)
	if err != nil {
		return "", fmt.Errorf("pubkey: bad private key: %w", err)
	}
	if len(seed) != ed25519.SeedSize {
		return "", errors.New("pubkey: private key wrong size")
	}
	priv := ed25519.NewKeyFromSeed(seed)
	sig := ed25519.Sign(priv, []byte(nonce))
	return base64.StdEncoding.EncodeToString(sig), nil
}

// VerifyChallenge checks a signed nonce against a stored public key.
func VerifyChallenge(pubKeyStr, nonce, sigB64 string) error {
	raw, err := decodePubKey(pubKeyStr)
	if err != nil {
		return err
	}
	if len(raw) != ed25519.PublicKeySize {
		return errors.New("pubkey: public key wrong size")
	}
	sig, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return fmt.Errorf("pubkey: bad signature: %w", err)
	}
	if !ed25519.Verify(raw, []byte(nonce), sig) {
		return errors.New("pubkey: signature mismatch")
	}
	return nil
}

// decodePubKey accepts "ssh-ed25519 BASE64" or raw BASE64.
func decodePubKey(s string) (ed25519.PublicKey, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "ssh-ed25519 ")
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("pubkey: decode: %w", err)
	}
	return ed25519.PublicKey(b), nil
}

func randUint64() uint64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
}

// AdminKeyBundle is the on-disk format for locally held admin private keys.
// Never commit this file — it holds secrets. .gitignore excludes *.json admin keys.
type AdminKeyBundle struct {
	Accounts map[string]KeyPair `json:"accounts"`
}

// LoadAdminKeys reads locally held admin private keys (gitignored file).
func LoadAdminKeys(path string) (*AdminKeyBundle, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var bundle AdminKeyBundle
	if err := json.Unmarshal(b, &bundle); err != nil {
		return nil, err
	}
	if bundle.Accounts == nil {
		bundle.Accounts = map[string]KeyPair{}
	}
	return &bundle, nil
}

// SaveAdminKeys writes locally held admin private keys (gitignored file).
func SaveAdminKeys(path string, bundle *AdminKeyBundle) error {
	b, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}
