package link

import "testing"

func TestGetSizeTextCountsBytesBelowOneMegabyte(t *testing.T) {
	if got := GetSizeText((1 << 20) - 1); got != "1048575 bytes" {
		t.Errorf("GetSizeText(1MB-1) = %q, want %q", got, "1048575 bytes")
	}
	if got := GetSizeText(1 << 20); got != "1 MB" {
		t.Errorf("GetSizeText(1MB) = %q, want %q", got, "1 MB")
	}
}
