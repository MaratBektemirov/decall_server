package auth

import (
	"encoding/binary"
	"errors"
)

// Minimal COSE EC2 (ES256 / P-256) → PKIX SPKI converter for WebAuthn credentials.
func cosePublicKeyToSPKI(coseKey []byte) ([]byte, error) {
	m, err := decodeCBORMap(coseKey)
	if err != nil {
		return nil, err
	}

	kty, _ := m[1].(int64)
	alg, _ := m[3].(int64)
	crv, _ := m[-1].(int64)
	x, _ := m[-2].([]byte)
	y, _ := m[-3].([]byte)

	if kty != 2 || alg != -7 || crv != 1 {
		return nil, errors.New("unsupported COSE key")
	}
	if len(x) != 32 || len(y) != 32 {
		return nil, errors.New("invalid COSE coordinates")
	}

	point := make([]byte, 1+len(x)+len(y))
	point[0] = 0x04
	copy(point[1:], x)
	copy(point[1+len(x):], y)

	algorithmID := []byte{
		0x30, 0x13, 0x06, 0x07, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x02, 0x01, 0x06, 0x08, 0x2a, 0x86, 0x48, 0xce, 0x3d,
		0x03, 0x01, 0x07,
	}

	bitString := make([]byte, 3+len(point))
	bitString[0] = 0x03
	bitString[1] = byte(len(point) + 1)
	bitString[2] = 0x00
	copy(bitString[3:], point)

	spki := make([]byte, 2+len(algorithmID)+len(bitString))
	spki[0] = 0x30
	spki[1] = byte(len(algorithmID) + len(bitString))
	copy(spki[2:], algorithmID)
	copy(spki[2+len(algorithmID):], bitString)

	return spki, nil
}

func decodeCBORMap(data []byte) (map[int64]any, error) {
	val, _, err := decodeCBORValue(data, 0)
	if err != nil {
		return nil, err
	}
	m, ok := val.(map[int64]any)
	if !ok {
		return nil, errors.New("expected CBOR map")
	}
	return m, nil
}

func decodeCBORValue(data []byte, offset int) (any, int, error) {
	if offset >= len(data) {
		return nil, offset, errors.New("unexpected end of CBOR")
	}
	initial := data[offset]
	major := initial >> 5
	additional := initial & 0x1f
	next := offset + 1

	switch major {
	case 0:
		length, end, err := readCBORLength(data, next, additional)
		if err != nil {
			return nil, offset, err
		}
		return int64(length), end, nil
	case 2:
		length, start, err := readCBORLength(data, next, additional)
		if err != nil {
			return nil, offset, err
		}
		end := start + length
		if end > len(data) {
			return nil, offset, errors.New("invalid CBOR byte string")
		}
		out := make([]byte, length)
		copy(out, data[start:end])
		return out, end, nil
	case 5:
		length, start, err := readCBORLength(data, next, additional)
		if err != nil {
			return nil, offset, err
		}
		m := make(map[int64]any, length)
		cursor := start
		for i := 0; i < length; i++ {
			keyAny, keyEnd, err := decodeCBORValue(data, cursor)
			if err != nil {
				return nil, offset, err
			}
			key, ok := keyAny.(int64)
			if !ok {
				return nil, offset, errors.New("COSE map key must be integer")
			}
			val, valEnd, err := decodeCBORValue(data, keyEnd)
			if err != nil {
				return nil, offset, err
			}
			m[key] = val
			cursor = valEnd
		}
		return m, cursor, nil
	default:
		return nil, offset, errors.New("unsupported CBOR type")
	}
}

func readCBORLength(data []byte, offset int, additional byte) (int, int, error) {
	switch {
	case additional < 24:
		return int(additional), offset, nil
	case additional == 24:
		if offset >= len(data) {
			return 0, offset, errors.New("invalid CBOR length")
		}
		return int(data[offset]), offset + 1, nil
	case additional == 25:
		if offset+1 >= len(data) {
			return 0, offset, errors.New("invalid CBOR length")
		}
		return int(binary.BigEndian.Uint16(data[offset : offset+2])), offset + 2, nil
	case additional == 26:
		if offset+3 >= len(data) {
			return 0, offset, errors.New("invalid CBOR length")
		}
		return int(binary.BigEndian.Uint32(data[offset : offset+4])), offset + 4, nil
	default:
		return 0, offset, errors.New("unsupported CBOR length")
	}
}
