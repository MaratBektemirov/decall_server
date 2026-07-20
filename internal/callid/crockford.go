package callid

import (
	"strings"
	"unicode"
)

// Crockford Base32 alphabet (excludes I, L, O, U).
const crockfordAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

const encodedLength = 15

func encodeCrockford(data []byte, length int) string {
	var out strings.Builder
	out.Grow(length)

	bitBuffer := 0
	bitsInBuffer := 0
	byteIdx := 0

	for out.Len() < length {
		for bitsInBuffer < 5 && byteIdx < len(data) {
			bitBuffer = (bitBuffer << 8) | int(data[byteIdx])
			bitsInBuffer += 8
			byteIdx++
		}

		if bitsInBuffer < 5 {
			bitBuffer <<= 5 - bitsInBuffer
			bitsInBuffer = 5
		}

		index := (bitBuffer >> (bitsInBuffer - 5)) & 0x1F
		bitsInBuffer -= 5
		out.WriteByte(crockfordAlphabet[index])
	}

	return out.String()
}

func formatGrouped(raw string, groupSize int) string {
	if groupSize <= 0 || len(raw) == 0 {
		return raw
	}

	var b strings.Builder
	b.Grow(len(raw) + len(raw)/groupSize)

	for i, c := range raw {
		if i > 0 && i%groupSize == 0 {
			b.WriteByte('-')
		}
		b.WriteRune(c)
	}

	return b.String()
}

func decodeCrockfordRune(r rune) (byte, bool) {
	switch r {
	case 'o', 'O':
		r = '0'
	case 'i', 'I', 'l', 'L':
		r = '1'
	case 'u', 'U':
		r = 'V'
	default:
		r = unicode.ToUpper(r)
	}

	for i := 0; i < len(crockfordAlphabet); i++ {
		if rune(crockfordAlphabet[i]) == r {
			return crockfordAlphabet[i], true
		}
	}

	return 0, false
}
