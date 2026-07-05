package words

import (
	"errors"
	"fmt"
	"strings"
)

// NormalizeCallID strips non-digits and formats as XXXX-XXXX-XXXX-XXXX.
func NormalizeCallID(id string) (string, error) {
	var digits strings.Builder
	for _, r := range id {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}

	d := digits.String()
	if len(d) != 16 {
		return "", errors.New("call id must contain 16 digits")
	}

	return fmt.Sprintf("%s-%s-%s-%s", d[0:4], d[4:8], d[8:12], d[12:16]), nil
}
