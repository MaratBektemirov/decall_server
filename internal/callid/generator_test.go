package callid

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestGenerateID(t *testing.T) {
	const numKeys = 500_000

	seenIDs := make(map[string]struct{})

	for i := 0; i < numKeys; i++ {
		addrBytes := make([]byte, 20)
		_, err := rand.Read(addrBytes)
		if err != nil {
			t.Fatalf("failed to generate random bytes: %v", err)
		}

		pubKeyStr := fmt.Sprintf("secp256k1|hex|0x%s", hex.EncodeToString(addrBytes))

		callID := GenerateID(pubKeyStr)

		if _, exists := seenIDs[callID]; exists {
			t.Fatalf("COLLISION DETECTED! Call ID %s generated for multiple keys on iteration %d\nConflicting Input: %s", callID, i, pubKeyStr)
		}

		seenIDs[callID] = struct{}{}
	}

	t.Logf("Successfully generated %d unique IDs with 0 collisions", numKeys)
}
