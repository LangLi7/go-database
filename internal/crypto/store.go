package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// keyRecord is the serialized form of an EncryptionKey with encrypted private key
type keyRecord struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	Algorithm  Algorithm  `json:"algorithm"`
	Purpose    KeyPurpose `json:"purpose"`
	PublicKey  []byte     `json:"public_key,omitempty"`
	EncPrivKey string     `json:"enc_priv_key"` // AES-GCM encrypted base64
	CreatedAt  int64      `json:"created_at"`
	ExpiresAt  int64      `json:"expires_at,omitempty"`
	Version    int        `json:"version"`
}

// KeyStore persists encryption keys to an encrypted JSON file
type KeyStore struct {
	mu     sync.RWMutex
	path   string
	master []byte // 32-byte master key for encrypting private keys
	keys   map[string]*EncryptionKey
}

// NewKeyStore creates a store, loading existing keys
func NewKeyStore(path string, masterKey []byte) (*KeyStore, error) {
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("master key must be 32 bytes")
	}
	s := &KeyStore{
		path:   path,
		master: masterKey,
		keys:   make(map[string]*EncryptionKey),
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("reading keystore: %w", err)
	}
	var records []keyRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("parsing keystore: %w", err)
	}
	for _, rec := range records {
		var privKey []byte
		if rec.EncPrivKey != "" {
			privKey, err = s.decryptKey(rec.EncPrivKey)
			if err != nil {
				return nil, fmt.Errorf("decrypt key %s: %w", rec.ID, err)
			}
		}
		s.keys[rec.ID] = &EncryptionKey{
			ID:        rec.ID,
			UserID:    rec.UserID,
			Algorithm: rec.Algorithm,
			Purpose:   rec.Purpose,
			PublicKey: rec.PublicKey,
			PrivKey:   privKey,
			CreatedAt: rec.CreatedAt,
			ExpiresAt: rec.ExpiresAt,
			Version:   rec.Version,
		}
	}
	return s, nil
}

func (s *KeyStore) persist() error {
	// Caller must hold s.mu (write lock). Do NOT lock again — would deadlock.
	records := make([]keyRecord, 0, len(s.keys))
	for _, k := range s.keys {
		encPriv := ""
		if k.PrivKey != nil {
			var err error
			encPriv, err = s.encryptKey(k.PrivKey)
			if err != nil {
				return fmt.Errorf("encrypt key %s: %w", k.ID, err)
			}
		}
		records = append(records, keyRecord{
			ID:         k.ID,
			UserID:     k.UserID,
			Algorithm:  k.Algorithm,
			Purpose:    k.Purpose,
			PublicKey:  k.PublicKey,
			EncPrivKey: encPriv,
			CreatedAt:  k.CreatedAt,
			ExpiresAt:  k.ExpiresAt,
			Version:    k.Version,
		})
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func (s *KeyStore) encryptKey(privKey []byte) (string, error) {
	block, err := aes.NewCipher(s.master)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	out := gcm.Seal(nil, nonce, privKey, nil)
	return encodeBase64(append(nonce, out...)), nil
}

func (s *KeyStore) decryptKey(enc string) ([]byte, error) {
	data, err := decodeBase64(enc)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(s.master)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("encrypted key too short")
	}
	return gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
}

func (s *KeyStore) List(userID string) ([]*EncryptionKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*EncryptionKey
	for _, k := range s.keys {
		if userID == "" || k.UserID == userID {
			cp := *k
			cp.PrivKey = nil
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (s *KeyStore) Get(id string) (*EncryptionKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	k, ok := s.keys[id]
	if !ok {
		return nil, fmt.Errorf("key %s not found", id)
	}
	cp := *k
	cp.PrivKey = make([]byte, len(k.PrivKey))
	copy(cp.PrivKey, k.PrivKey)
	return &cp, nil
}

func (s *KeyStore) Save(key *EncryptionKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *key
	s.keys[key.ID] = &cp
	return s.persist()
}

func (s *KeyStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.keys, id)
	return s.persist()
}

// GenKeyID generates a unique key identifier
func GenKeyID() string {
	return fmt.Sprintf("key-%d", time.Now().UnixNano())
}
