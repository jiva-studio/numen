package identity

import (
	"crypto/rand"
	"errors"
	"testing"
	"time"
)

func TestAdopt_RefusesARevokeWhoseKeyHistoryHasAnotherBoxThanTheEpochBefore(t *testing.T) {
	t.Parallel()
	f := newForgery(t)
	state := f.newOtherState(t)
	stranger, err := NewAccountKeys(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	last := f.chain.entries[len(f.chain.entries)-1].Time
	base := Chain{entries: f.chain.entries[:len(f.chain.entries)-1]}

	forged, _, err := base.RevokeDevice(f.signer, stranger, f.revoked.Sign, rand.Reader, last.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := state.Adopt(forged); !errors.Is(err, ErrWrapMismatch) {
		t.Errorf("Adopt() = %v, want ErrWrapMismatch", err)
	}
}
