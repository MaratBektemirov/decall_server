package auth

// VerifyProofOptions carries extra data required for some key types.
type VerifyProofOptions struct {
	WebAuthnCredentialPublicKey string `json:"credentialPublicKey,omitempty"`
	ExpectedOrigin              string `json:"expectedOrigin,omitempty"`
}
