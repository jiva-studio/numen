//go:build linux || darwin

package process

import (
	"errors"
	"testing"

	"golang.org/x/sys/unix"
)

func TestARefusedPriorityChangeIsADeniedNice(t *testing.T) {
	for _, refusal := range []error{unix.EACCES, unix.EPERM} {
		err := markNiceDenied(refusal)

		if !errors.Is(err, ErrNiceDenied) || !errors.Is(err, refusal) {
			t.Errorf("markNiceDenied(%v) = %v, want ErrNiceDenied wrapping the refusal", refusal, err)
		}
	}
}

func TestAnotherPriorityFailureIsNotADeniedNice(t *testing.T) {
	err := markNiceDenied(unix.EINVAL)

	if errors.Is(err, ErrNiceDenied) || !errors.Is(err, unix.EINVAL) {
		t.Errorf("markNiceDenied(EINVAL) = %v, want the error unchanged", err)
	}
}
