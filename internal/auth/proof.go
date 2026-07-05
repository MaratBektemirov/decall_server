package auth

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"decall_server/internal/config"
)

const secretAuthHeader = " wants you to prove your signing key:"

var (
	nonceLineRe = regexp.MustCompile(`(?m)^nonce:\s*(.+)$`)
	expLineRe   = regexp.MustCompile(`(?m)^exp:\s*(\d+)$`)
)

type SecretAuthProof struct {
	Message   string              `json:"message"`
	Signature secretAuthSignature `json:"signature"`
	PubKey    secretAuthPubKey    `json:"pubKey"`
}

type secretAuthSignature struct {
	Value     string `json:"value"`
	Encoding  string `json:"encoding,omitempty"`
	Extension *struct {
		AuthenticatorData string `json:"authenticatorData"`
		ClientDataJSON    string `json:"clientDataJSON"`
	} `json:"extension,omitempty"`
}

type secretAuthPubKey struct {
	Algorithm string `json:"algorithm"`
	Source    string `json:"source"`
	Value     string `json:"value"`
	Encoding  string `json:"encoding"`
}

func (h *Handler) VerifyProof(cfg config.Config, proof SecretAuthProof) error {
	if strings.TrimSpace(proof.Message) == "" {
		return errors.New("proof message required")
	}

	challenge, err := parseChallengeMessage(proof.Message)
	if err != nil {
		return err
	}

	domain := cfg.AuthDomain
	if domain == "" {
		domain = "localhost"
	}
	if challenge.Domain != domain {
		return fmt.Errorf("domain mismatch")
	}

	now := time.Now().Unix()
	if challenge.Exp <= now {
		return errors.New("challenge expired")
	}

	if !h.nonceIssued(challenge.Nonce, challenge.Exp) {
		return errors.New("unknown or expired nonce")
	}

	if proof.PubKey.Algorithm == "" || proof.PubKey.Source == "" || proof.PubKey.Value == "" {
		return errors.New("invalid pubKey")
	}
	if proof.Signature.Value == "" {
		return errors.New("invalid signature")
	}
	if proof.Signature.Extension != nil {
		return errors.New("webauthn proofs are not supported yet")
	}

	switch {
	case proof.PubKey.Algorithm == "secp256k1" && proof.PubKey.Source == "ethereum":
		if !verifyEthereumPersonalSign(proof.Message, proof.Signature.Value, proof.PubKey.Value) {
			return errors.New("invalid signature")
		}
	case proof.PubKey.Algorithm == "secp256k1" && proof.PubKey.Source == "raw":
		if !verifyRawSecp256k1PersonalSign(proof.Message, proof.Signature.Value, proof.PubKey.Value, signatureEncoding(proof)) {
			return errors.New("invalid signature")
		}
	default:
		return fmt.Errorf("unsupported key type: %s/%s", proof.PubKey.Algorithm, proof.PubKey.Source)
	}

	return nil
}

func (h *Handler) nonceIssued(nonce string, exp int64) bool {
	nonce = strings.TrimSpace(nonce)
	if nonce == "" {
		return false
	}

	v, ok := h.nonces.Load(nonce)
	if !ok {
		return false
	}

	storedExp, ok := v.(int64)
	if !ok || storedExp != exp {
		return false
	}

	return time.Now().Unix() < storedExp
}

type parsedChallenge struct {
	Domain string
	Nonce  string
	Exp    int64
}

func parseChallengeMessage(message string) (parsedChallenge, error) {
	headerIndex := strings.Index(message, secretAuthHeader)
	if headerIndex < 1 {
		return parsedChallenge{}, errors.New("invalid challenge message")
	}

	domain := strings.TrimSpace(message[:headerIndex])
	if domain == "" {
		return parsedChallenge{}, errors.New("invalid challenge domain")
	}

	nonceMatch := nonceLineRe.FindStringSubmatch(message)
	expMatch := expLineRe.FindStringSubmatch(message)
	if len(nonceMatch) < 2 || len(expMatch) < 2 {
		return parsedChallenge{}, errors.New("invalid challenge message")
	}

	nonce := strings.TrimSpace(nonceMatch[1])
	exp, err := parseExp(expMatch[1])
	if err != nil {
		return parsedChallenge{}, err
	}
	if nonce == "" {
		return parsedChallenge{}, errors.New("invalid challenge nonce")
	}

	return parsedChallenge{Domain: domain, Nonce: nonce, Exp: exp}, nil
}

func parseExp(value string) (int64, error) {
	var exp int64
	for _, ch := range strings.TrimSpace(value) {
		if ch < '0' || ch > '9' {
			return 0, errors.New("invalid challenge exp")
		}
		exp = exp*10 + int64(ch-'0')
	}
	return exp, nil
}

func signatureEncoding(proof SecretAuthProof) string {
	if enc := strings.TrimSpace(proof.Signature.Encoding); enc != "" {
		return enc
	}
	return pubKeyEncoding(proof.PubKey)
}

func pubKeyEncoding(key secretAuthPubKey) string {
	if enc := strings.TrimSpace(key.Encoding); enc != "" {
		return enc
	}
	return "hex"
}

func verifyEthereumPersonalSign(message, signature, address string) bool {
	sig, err := decodeSecp256k1Signature(signature, "hex")
	if err != nil {
		return false
	}

	hash := accounts.TextHash([]byte(message))
	pubKey, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return false
	}

	recovered := crypto.PubkeyToAddress(*pubKey)
	want := common.HexToAddress(address)
	return strings.EqualFold(recovered.Hex(), want.Hex())
}

func verifyRawSecp256k1PersonalSign(message, signature, pubKeyValue, encoding string) bool {
	sig, err := decodeSecp256k1Signature(signature, encoding)
	if err != nil {
		return false
	}

	hash := accounts.TextHash([]byte(message))
	recoveredPub, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return false
	}

	want, err := decodeSecp256k1PublicKey(pubKeyValue)
	if err != nil {
		return false
	}

	got := crypto.FromECDSAPub(recoveredPub)
	return hex.EncodeToString(got) == hex.EncodeToString(want)
}

func decodeSecp256k1Signature(value, encoding string) ([]byte, error) {
	sig, err := decodeBytes(value, encoding)
	if err != nil {
		return nil, err
	}
	if len(sig) != 65 {
		return nil, errors.New("invalid secp256k1 signature length")
	}
	if sig[64] >= 27 {
		sig[64] -= 27
	}
	return sig, nil
}

func decodeSecp256k1PublicKey(value string) ([]byte, error) {
	hexValue := strings.TrimPrefix(strings.TrimPrefix(value, "0x"), "0X")
	if len(hexValue) != 66 && len(hexValue) != 130 {
		return nil, errors.New("invalid secp256k1 public key")
	}
	return hex.DecodeString(hexValue)
}

func decodeBytes(value, encoding string) ([]byte, error) {
	value = strings.TrimSpace(value)
	switch strings.ToLower(encoding) {
	case "hex":
		if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
			return hexutil.Decode(value)
		}
		return hex.DecodeString(value)
	case "base64":
		return base64.StdEncoding.DecodeString(value)
	case "base64url":
		return base64.RawURLEncoding.DecodeString(value)
	default:
		return nil, fmt.Errorf("unsupported encoding: %s", encoding)
	}
}
