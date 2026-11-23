package fakeserver_test

import (
	"bytes"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
)

func TestRestore_ReadsWhatSnapshotKept(t *testing.T) {
	first := fakeserver.New(fakeserver.Config{})
	kept, err := first.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	second := fakeserver.New(fakeserver.Config{})

	if err := second.Restore(kept); err != nil {
		t.Fatal(err)
	}

	again, err := second.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(kept, again) {
		t.Errorf("snapshot after restore = %s, want %s", again, kept)
	}
}

func TestRestore_RefusesWhatIsNotASnapshot(t *testing.T) {
	if err := fakeserver.New(fakeserver.Config{}).Restore([]byte("not json")); err == nil {
		t.Error("Restore of text that is not a snapshot succeeded, want an error")
	}
}
