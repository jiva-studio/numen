package keychain

import (
	"fmt"
	"sync"
	"testing"
)

func TestTwoStoresOnOneFileKeepEveryEntryBothWrite(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	editor, flashcards := newTestStore(dir), newTestStore(dir)
	const each = 40

	var group sync.WaitGroup
	for i := range each {
		group.Add(2)
		go func() {
			defer group.Done()
			if err := editor.SetKey(t.Context(), fmt.Sprintf("secret/a%07d", i), "value-a"); err != nil {
				t.Error(err)
			}
		}()
		go func() {
			defer group.Done()
			if err := flashcards.SetKey(t.Context(), fmt.Sprintf("secret/b%07d", i), "value-b"); err != nil {
				t.Error(err)
			}
		}()
	}
	group.Wait()

	if got := len(readCredentials(t, dir)); got != 2*each {
		t.Errorf("the file holds %d entries, want %d", got, 2*each)
	}
}
