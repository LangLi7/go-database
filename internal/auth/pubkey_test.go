package auth

import (
	"testing"
)

func TestPubKeyRoundTrip(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if kp.PublicKey == "" || kp.PrivateKey == "" {
		t.Fatal("keypair empty")
	}

	nonce := "abc123nonce"
	sig, err := SignChallenge(kp.PrivateKey, nonce)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if err := VerifyChallenge(kp.PublicKey, nonce, sig); err != nil {
		t.Fatalf("verify valid sig: %v", err)
	}

	// wrong nonce must fail
	if err := VerifyChallenge(kp.PublicKey, "different-nonce", sig); err == nil {
		t.Fatal("verify should fail for wrong nonce")
	}

	// tampered signature must fail
	if err := VerifyChallenge(kp.PublicKey, nonce, sig+"x"); err == nil {
		t.Fatal("verify should fail for tampered signature")
	}
}

func TestChallengeExpiryWindow(t *testing.T) {
	ch := NewChallenge("alice")
	if ch.ExpiresAt <= 0 {
		t.Fatal("challenge has no expiry")
	}
	if ch.Nonce == "" {
		t.Fatal("challenge nonce empty")
	}
}
