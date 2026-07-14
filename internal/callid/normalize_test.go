package callid

import "testing"

func TestNormalizeCallID(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"1234-5678-9012-3456", "1234-5678-9012-3456"},
		{"1234567890123456", "1234-5678-9012-3456"},
		{"1234 5678 9012 3456", "1234-5678-9012-3456"},
		{"1234.5678.9012.3456", "1234-5678-9012-3456"},
	}

	for _, tc := range tests {
		got, err := Normalize(tc.in)
		if err != nil {
			t.Fatalf("NormalizeCallID(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("NormalizeCallID(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeCallIDInvalid(t *testing.T) {
	if _, err := Normalize("1234-5678"); err == nil {
		t.Fatal("expected error for short id")
	}
}
