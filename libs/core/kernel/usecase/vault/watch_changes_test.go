package vault_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
)

type pushedWatcher struct {
	changes chan []string
	loss    chan struct{}
	err     error
	watched domain.Vault
}

func (w *pushedWatcher) WatchVault(_ context.Context, v domain.Vault) (<-chan []string, <-chan struct{}, error) {
	w.watched = v
	return w.changes, w.loss, w.err
}

func TestTheChangesOfAWatchedVaultAreHandedOnWithTheLossOfTheWatch(t *testing.T) {
	watcher := &pushedWatcher{changes: make(chan []string, 1), loss: make(chan struct{})}
	u := vault.NewWatchChanges(watcher)

	changes, loss, err := u.Execute(t.Context(), domain.Vault{ID: "v1", Path: "/vaults/Rivers"})
	if err != nil {
		t.Fatal(err)
	}
	watcher.changes <- []string{"Danube.md"}
	close(watcher.loss)

	if got := <-changes; len(got) != 1 || got[0] != "Danube.md" {
		t.Errorf("the change was %v", got)
	}
	<-loss
	if watcher.watched.ID != "v1" {
		t.Errorf("the vault watched was %+v", watcher.watched)
	}
}

func TestAWatchThatCannotBeStartedReturnsWhyNotNoChannels(t *testing.T) {
	failure := errors.New("the folder is gone")
	u := vault.NewWatchChanges(&pushedWatcher{err: failure})

	_, _, err := u.Execute(t.Context(), domain.Vault{ID: "v1"})

	if !errors.Is(err, failure) {
		t.Errorf("the watch was %v, want the reason it did not start", err)
	}
}
