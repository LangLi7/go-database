package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"math/big"
)

// x509MarshalPKIX marshals an EC public key to PKIX DER.
func x509MarshalPKIX(pub *ecdsa.PublicKey) ([]byte, error) {
	return x509.MarshalPKIXPublicKey(pub)
}

// x509MarshalECPrivate marshals an EC private key to PKCS8 DER.
func x509MarshalECPrivate(priv *ecdsa.PrivateKey) ([]byte, error) {
	return x509.MarshalPKCS8PrivateKey(priv)
}

// x509ParseECPrivate parses a PKCS8 DER EC private key.
func x509ParseECPrivate(der []byte) (*ecdsa.PrivateKey, error) {
	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("parse ec priv: %w", err)
	}
	priv, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an EC private key")
	}
	return priv, nil
}

// x509ParsePKIXPublic parses a PKIX DER public key.
func x509ParsePKIXPublic(der []byte) (any, error) {
	return x509.ParsePKIXPublicKey(der)
}

// bytesReader returns a reader over a byte slice (avoids io.NopCloser import churn).
func bytesReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}

// GenerateECDSAKeypair is a convenience used by tests/helpers if needed.
func GenerateECDSAKeypair() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// bigIntFromBytes is a tiny helper kept for signature parsing symmetry.
func bigIntFromBytes(b []byte) *big.Int {
	return new(big.Int).SetBytes(b)
}

var _ = hex.EncodeToString
