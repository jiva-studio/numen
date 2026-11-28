package identity

import (
	"errors"
	"testing"
)

func TestAdopt_RefusesARevokeWhoseAccountBoxIsNotTheOneWrappedForTheDevice(t *testing.T) {
	t.Parallel()
	f := newForgery(t)
	state := f.newOtherState(t)
	forged := f.resign(func(e *Entry) { e.Revoke.AccountBox = [32]byte{9} })

	if _, err := state.Adopt(forged); !errors.Is(err, ErrWrapMismatch) {
		t.Errorf("Adopt() = %v, want ErrWrapMismatch", err)
	}
}

func TestAdopt_RevokeWithUnsealedKeyHistoryRejected(t *testing.T) {
	t.Parallel()
	f := newForgery(t)
	state := f.newOtherState(t)
	forged := f.resign(func(e *Entry) { e.Revoke.PreviousKeys = make([]byte, previousKeysSize) })

	if _, err := state.Adopt(forged); !errors.Is(err, ErrWrapMismatch) {
		t.Errorf("Adopt() = %v, want ErrWrapMismatch", err)
	}
}

func TestAdopt_AcceptsTheRevokeThatTheForgeryStartsFrom(t *testing.T) {
	t.Parallel()
	f := newForgery(t)
	state := f.newOtherState(t)

	if _, err := state.Adopt(f.chain); err != nil {
		t.Errorf("Adopt() = %v, want the honest revoke adopted", err)
	}
}

func (f forgery) newOtherState(tb testing.TB) State {
	tb.Helper()
	wrapped, err := f.keys.WrapForDevice(f.other.Public())
	if err != nil {
		tb.Fatal(err)
	}
	state, err := NewState(Record{Device: f.other, Wrapped: wrapped}, Chain{entries: f.chain.entries[:len(f.chain.entries)-1]})
	if err != nil {
		tb.Fatal(err)
	}
	return state
}
