package auth

import (
	"crypto/sha256"
	"errors"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"
)

func verifyTronAddressSign(message, signature, address string) bool {
	sig, err := decodeSecp256k1Signature(signature, "hex")
	if err != nil {
		return false
	}

	expected, err := decodeTronAddressPayload(address)
	if err != nil {
		return false
	}

	hash := hashTronSignMessageV2([]byte(message))
	recoveredPub, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return false
	}

	uncompressed := crypto.FromECDSAPub(recoveredPub)
	if len(uncompressed) != 65 || uncompressed[0] != 4 {
		return false
	}

	recoveredPayload := crypto.Keccak256(uncompressed[1:])[12:]
	return constantTimeEqual(recoveredPayload, expected)
}

func hashTronSignMessageV2(content []byte) []byte {
	var digest []byte
	if len(content) == 32 {
		digest = content
	} else {
		sum := sha256.Sum256(content)
		digest = sum[:]
	}

	prefix := []byte("\x19TRON Signed Message:\n32")
	payload := append(prefix, digest...)
	return crypto.Keccak256(payload)
}

func decodeTronAddressPayload(address string) ([]byte, error) {
	decoded, err := base58.Decode(address)
	if err != nil || len(decoded) != 25 || decoded[0] != 0x41 {
		return nil, errors.New("invalid tron address")
	}
	return decoded[1:21], nil
}
