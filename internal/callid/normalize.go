package callid

import (
	"errors"
	"strings"
)

// Normalize strips separators, applies Crockford decoding rules, and formats as XXXXX-XXXXX-XXXXX.
func Normalize(id string) (string, error) {
	var chars strings.Builder
	chars.Grow(encodedLength)

	for _, r := range id {
		if r == '-' || r == ' ' || r == '.' {
			continue
		}

		c, ok := decodeCrockfordRune(r)
		if !ok {
			continue
		}

		chars.WriteByte(c)
	}

	raw := chars.String()
	if len(raw) != encodedLength {
		return "", errors.New("call id must contain 15 crockford characters")
	}

	return formatGrouped(raw, 5), nil
}
