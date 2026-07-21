package auth

import (
	"encoding/base64"
	"encoding/hex"

	"github.com/go-webauthn/webauthn/webauthn"
)

// Passkey is a stored WebAuthn credential for a user (Windows Hello / TouchID /
// Android / security key). Stored separately from User so SaveUser's
// INSERT OR REPLACE never wipes credentials.
type Passkey struct {
	ID           string `json:"id"`          // credential ID (base64)
	UserID       string `json:"user_id"`     // owner (auth.User.ID)
	Name         string `json:"name"`        // human label, e.g. "Laptop TPM"
	PublicKey    []byte `json:"-"`           // COSE key, never serialized to client
	CredentialID []byte `json:"-"`           // raw credential ID
	Attestation  string `json:"attestation"` // attestation type
	AAGUID       string `json:"aaguid"`      // authenticator AAGUID (hex)
	SignCount    uint32 `json:"sign_count"`  // anti-replay counter
	CreatedAt    int64  `json:"created_at"`
}

// webauthnUser adapts auth.User + its passkeys to the webauthn.User interface.
type WebAuthnUser struct {
	User        *User
	Credentials []webauthn.Credential
}

func (w *WebAuthnUser) WebAuthnID() []byte                         { return []byte(w.User.ID) }
func (w *WebAuthnUser) WebAuthnName() string                       { return w.User.Username }
func (w *WebAuthnUser) WebAuthnDisplayName() string                { return w.User.Username }
func (w *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential { return w.Credentials }

// ToCredential converts a stored Passkey into a webauthn.Credential.
func (p *Passkey) ToCredential() webauthn.Credential {
	return webauthn.Credential{
		ID:              p.CredentialID,
		PublicKey:       p.PublicKey,
		AttestationType: p.Attestation,
		Flags: webauthn.CredentialFlags{
			UserPresent:    true,
			UserVerified:   true,
			BackupEligible: false,
			BackupState:    false,
		},
		Authenticator: webauthn.Authenticator{
			AAGUID:    decodeHexOrEmpty(p.AAGUID),
			SignCount: p.SignCount,
		},
	}
}

// PasskeyFromCredential builds a Passkey from a webauthn.Credential after registration.
func PasskeyFromCredential(userID, name string, c webauthn.Credential, createdAt int64) *Passkey {
	return &Passkey{
		ID:           encodeBase64(c.ID),
		UserID:       userID,
		Name:         name,
		PublicKey:    c.PublicKey,
		CredentialID: c.ID,
		Attestation:  c.AttestationType,
		AAGUID:       hexEncode(c.Authenticator.AAGUID),
		SignCount:    c.Authenticator.SignCount,
		CreatedAt:    createdAt,
	}
}

// ─── minimal stdlib helpers (no external dep) ───
func encodeBase64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

func hexEncode(b []byte) string { return hex.EncodeToString(b) }

func decodeHexOrEmpty(s string) []byte {
	if s == "" {
		return []byte{}
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return []byte{}
	}
	return b
}
