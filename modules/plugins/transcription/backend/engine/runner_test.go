package engine

import "testing"

// A status of zero reports no error.
func TestCheckStatusZeroIsNil(t *testing.T) {
	if err := checkStatus(0); err != nil {
		t.Errorf("checkStatus(0) = %v, want nil", err)
	}
}
