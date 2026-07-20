package callid

import "crypto/sha256"

func GenerateID(pubKey string) string {
	hash := sha256.Sum256([]byte(pubKey))
	raw := encodeCrockford(hash[:10], encodedLength)
	return formatGrouped(raw, 5)
}
