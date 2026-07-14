package callid

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

func GenerateID(pubKey string) string {
	hash := sha256.Sum256([]byte(pubKey))

	w1 := binary.BigEndian.Uint16(hash[0:2])
	w2 := binary.BigEndian.Uint16(hash[2:4])
	w3 := binary.BigEndian.Uint16(hash[4:6])
	w4 := binary.BigEndian.Uint16(hash[6:8])

	return fmt.Sprintf("%04d-%04d-%04d-%04d",
		w1%10000,
		w2%10000,
		w3%10000,
		w4%10000,
	)
}
