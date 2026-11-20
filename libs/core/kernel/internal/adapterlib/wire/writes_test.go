package wire

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
)

func TestWriteAdmissionIsScopedToTheWindow(t *testing.T) {
	first, second := NewWrites(nil), NewWrites(nil)
	finishFirst, err := first.AdmitWrite()
	if err != nil {
		t.Fatal(err)
	}
	drained := first.Seal()
	if isClosed(drained) {
		t.Fatal("the first window drained while its write was running")
	}
	if release, err := first.AdmitWrite(); !errors.Is(err, window.ErrClosing) || release != nil {
		t.Fatalf("sealed window admitted a write or returned the wrong error: %v", err)
	}
	finishSecond, err := second.AdmitWrite()
	if err != nil {
		t.Fatalf("second window refused a write: %v", err)
	}
	defer finishSecond()

	finishFirst()
	if !isClosed(drained) {
		t.Fatal("the first window still waited after its write finished")
	}
	first.Open()
	finishNext, err := first.AdmitWrite()
	if err != nil {
		t.Fatalf("the reopened window refused a write: %v", err)
	}
	finishNext()
}
