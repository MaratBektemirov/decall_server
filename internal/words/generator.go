package words

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/cosmos/go-bip39"
)

// GenerateID returns 4 word by BIP39 standard
func GenerateID(pubKey string) string {
	wordList := bip39.WordList
	dictLen := uint16(len(wordList))

	hash := sha256.Sum256([]byte(pubKey))

	w1 := binary.BigEndian.Uint16(hash[0:2])
	w2 := binary.BigEndian.Uint16(hash[2:4])
	w3 := binary.BigEndian.Uint16(hash[4:6])
	w4 := binary.BigEndian.Uint16(hash[6:8])

	return fmt.Sprintf("%s-%s-%s-%s",
		wordList[w1%dictLen],
		wordList[w2%dictLen],
		wordList[w3%dictLen],
		wordList[w4%dictLen],
	)
}
