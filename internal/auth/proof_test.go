package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"strconv"
	"testing"
	"time"

	"decall_server/internal/config"
)

func TestParseChallengeMessage(t *testing.T) {
	msg := "example.com wants you to prove your signing key:\n\nnonce: abc123\nexp: 1719667500"
	ch, err := parseChallengeMessage(msg)
	if err != nil {
		t.Fatalf("parseChallengeMessage: %v", err)
	}
	if ch.Domain != "example.com" || ch.Nonce != "abc123" || ch.Exp != 1719667500 {
		t.Fatalf("unexpected challenge: %+v", ch)
	}
}

func TestHandlerNonceIssued(t *testing.T) {
	h := NewHandler(config.Config{})
	exp := time.Now().Unix() + 60
	h.nonces.Store("nonce-1", exp)

	if !h.nonceIssued("nonce-1", exp) {
		t.Fatal("expected issued nonce")
	}
	if h.nonceIssued("nonce-1", exp+1) {
		t.Fatal("expected exp mismatch")
	}
	if h.nonceIssued("missing", exp) {
		t.Fatal("expected missing nonce")
	}
}

func TestVerifyProofDomainMismatch(t *testing.T) {
	h := NewHandler(config.Config{})
	exp := time.Now().Unix() + 60
	h.nonces.Store("abc", exp)

	cfg := config.Config{AuthDomain: "server.example"}
	proof := SecretAuthProof{
		Message: "other.example wants you to prove your signing key:\n\nnonce: abc\nexp: " + strconv.FormatInt(exp, 10),
		Signature: secretAuthSignature{Value: "0x00"},
		PubKey: secretAuthPubKey{
			Algorithm: "secp256k1",
			Source:    "ethereum",
			Value:     "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb",
			Encoding:  "hex",
		},
	}

	if err := h.VerifyProof(cfg, proof, VerifyProofOptions{}); err == nil {
		t.Fatal("expected domain mismatch error")
	}
}

func TestVerifyEd25519RawProof(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	exp := time.Now().Unix() + 60
	nonce := "ed25519-test-nonce"
	message := "localhost wants you to prove your signing key:\n\nnonce: " + nonce + "\nexp: " + strconv.FormatInt(exp, 10)
	signature := ed25519.Sign(priv, []byte(message))

	h := NewHandler(config.Config{})
	h.nonces.Store(nonce, exp)

	proof := SecretAuthProof{
		Message: message,
		Signature: secretAuthSignature{
			Value:    base64.StdEncoding.EncodeToString(signature),
			Encoding: "base64",
		},
		PubKey: secretAuthPubKey{
			Algorithm: "Ed25519",
			Source:    "raw",
			Value:     base64.StdEncoding.EncodeToString(pub),
			Encoding:  "base64",
		},
	}

	if err := h.VerifyProof(config.Config{AuthDomain: "localhost"}, proof, VerifyProofOptions{}); err != nil {
		t.Fatalf("VerifyProof: %v", err)
	}
}
