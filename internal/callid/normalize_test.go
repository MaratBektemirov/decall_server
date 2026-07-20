package callid

import (
	"strings"
	"testing"
)

func TestNormalizeCallID(t *testing.T) {
	raw := encodeCrockford([]byte("test pubkey material for normalize"), encodedLength)
	want := formatGrouped(raw, 5)

	tests := []struct {
		in   string
		want string
	}{
		{want, want},
		{strings.ReplaceAll(want, "-", ""), want},
		{strings.ReplaceAll(want, "-", " "), want},
		{strings.ReplaceAll(want, "-", "."), want},
	}

	for _, tc := range tests {
		got, err := Normalize(tc.in)
		if err != nil {
			t.Fatalf("Normalize(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("Normalize(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeCallIDCrockfordAliases(t *testing.T) {
	got, err := Normalize("ooooo-11111-vvvvv")
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	want, err := Normalize("00000-11111-VVVVV")
	if err != nil {
		t.Fatalf("Normalize want: %v", err)
	}
	if got != want {
		t.Fatalf("Normalize aliases = %q, want %q", got, want)
	}
}

func TestNormalizeCallIDInvalid(t *testing.T) {
	if _, err := Normalize("7K9MX-2NP4R"); err == nil {
		t.Fatal("expected error for short id")
	}
}

func TestGenerateIDDeterministic(t *testing.T) {
	pubKey := "secp256k1|hex|0x1234567890abcdef1234567890abcdef12345678"
	first := GenerateID(pubKey)
	second := GenerateID(pubKey)
	if first != second {
		t.Fatalf("GenerateID not deterministic: %q vs %q", first, second)
	}

	normalized, err := Normalize(first)
	if err != nil {
		t.Fatalf("Normalize generated id: %v", err)
	}
	if normalized != first {
		t.Fatalf("generated id not normalized: %q vs %q", first, normalized)
	}
}
