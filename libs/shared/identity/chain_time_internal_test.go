package identity

import (
	"crypto/rand"
	"errors"
	"testing"
	"time"
)

func TestVerify_RefusesAnEntryOlderThanTheEntryBefore(t *testing.T) {
	t.Parallel()
	f := newForgery(t)

	chain := f.resign(func(e *Entry) { e.Time = e.Time.Add(-time.Hour) })

	if err := chain.Verify(); !errors.Is(err, ErrEntryTime) {
		t.Errorf("error = %v, want ErrEntryTime", err)
	}
}

func TestAppendDevice_DatesAnEntryNoEarlierThanTheEntryBefore(t *testing.T) {
	t.Parallel()
	f := newForgery(t)
	last := f.chain.entries[len(f.chain.entries)-1].Time
	device, err := NewDeviceKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	next, err := f.chain.AppendDevice(f.signer, device.Public(), "Phone", last.Add(-time.Hour))

	if err != nil {
		t.Fatal(err)
	}
	if got := next.entries[len(next.entries)-1].Time; !got.Equal(last) {
		t.Errorf("entry time = %v, want %v", got, last)
	}
}
