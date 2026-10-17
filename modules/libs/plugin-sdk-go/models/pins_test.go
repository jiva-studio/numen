package models

import "testing"

func TestAPinnedAddressHasItsSum(t *testing.T) {
	pins := Pins{"https://example.org/a.onnx": "abc"}
	if got := pins.GetSum("https://example.org/a.onnx"); got != "abc" {
		t.Errorf("GetSum() = %q", got)
	}
}

func TestAnAddressNobodyPinnedHasNoSum(t *testing.T) {
	pins := Pins{"https://example.org/a.onnx": "abc"}
	if got := pins.GetSum("https://example.org/b.onnx"); got != "" {
		t.Errorf("GetSum() = %q", got)
	}
	if got := Pins(nil).GetSum("https://example.org/a.onnx"); got != "" {
		t.Errorf("GetSum() on no pins = %q", got)
	}
}
