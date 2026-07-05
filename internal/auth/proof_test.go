package auth

import (
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

	if err := h.VerifyProof(cfg, proof); err == nil {
		t.Fatal("expected domain mismatch error")
	}
}
