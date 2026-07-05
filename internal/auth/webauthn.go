package auth

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"strings"
)

func verifyWebAuthnProof(proof SecretAuthProof, opts VerifyProofOptions, expectedRPID string) bool {
	if proof.PubKey.Algorithm != "ES256" || proof.PubKey.Source != "webauthn" {
		return false
	}
	if opts.WebAuthnCredentialPublicKey == "" {
		return false
	}
	ext := proof.Signature.Extension
	if ext == nil || ext.AuthenticatorData == "" || ext.ClientDataJSON == "" {
		return false
	}

	spki, err := resolveCredentialPublicKeySPKI(opts.WebAuthnCredentialPublicKey)
	if err != nil {
		return false
	}

	pub, err := x509.ParsePKIXPublicKey(spki)
	if err != nil {
		return false
	}
	ecKey, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return false
	}

	clientDataBytes, err := decodeBytes(ext.ClientDataJSON, "base64url")
	if err != nil {
		return false
	}
	var clientData struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
		Origin    string `json:"origin"`
	}
	if err := json.Unmarshal(clientDataBytes, &clientData); err != nil {
		return false
	}
	if clientData.Type != "webauthn.get" {
		return false
	}

	expectedChallenge := base64.RawURLEncoding.EncodeToString([]byte(proof.Message))
	if clientData.Challenge != expectedChallenge {
		return false
	}

	if opts.ExpectedOrigin != "" && clientData.Origin != opts.ExpectedOrigin {
		return false
	}

	authData, err := decodeBytes(ext.AuthenticatorData, "base64url")
	if err != nil || len(authData) < 37 {
		return false
	}
	if authData[32]&0x01 != 0x01 {
		return false
	}

	if expectedRPID != "" {
		rpIDHash := authData[5:37]
		expectedHash := sha256.Sum256([]byte(expectedRPID))
		if !constantTimeEqual(rpIDHash, expectedHash[:]) {
			return false
		}
	}

	sig, err := decodeBytes(proof.Signature.Value, signatureEncodingFor(proof.Signature))
	if err != nil {
		return false
	}

	clientDataHash := sha256.Sum256(clientDataBytes)
	signed := make([]byte, len(authData)+len(clientDataHash))
	copy(signed, authData)
	copy(signed[len(authData):], clientDataHash[:])

	digest := sha256.Sum256(signed)
	return ecdsa.VerifyASN1(ecKey, digest[:], sig)
}

func signatureEncodingFor(sig secretAuthSignature) string {
	if enc := strings.TrimSpace(sig.Encoding); enc != "" {
		return enc
	}
	return "base64url"
}

func constantTimeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

func resolveCredentialPublicKeySPKI(value string) ([]byte, error) {
	key, err := decodeBytes(value, "base64url")
	if err != nil {
		return nil, err
	}
	if len(key) > 0 && key[0] == 0x30 {
		return key, nil
	}
	spki, err := cosePublicKeyToSPKI(key)
	if err != nil {
		return nil, err
	}
	return spki, nil
}
