package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/service"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/deletions"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/syncwire"
)

func newPausedBackend(t *testing.T) (*service.Backend, *world) {
	t.Helper()
	w := newWorld(t)
	w.c.Connect(t, "Test")
	paused := transfer.PausedRemovals{
		Direction: transfer.DirectionSend, Files: []transfer.Removal{{FileID: []byte{1}, Path: "a.pdf"}, {FileID: []byte{2}, Path: "b.pdf"}},
		Tracked: 4, CreatedAt: time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC),
	}
	if err := w.c.State.WritePending(t.Context(), "Test", paused); err != nil {
		t.Fatal(err)
	}
	backend := service.NewBackend(w.c.SyncDeps()).WithDeletions(w.c.State, syncwire.NewResolveDeletions(w.c.WireDeps()))
	return backend, w
}

func TestListVaults_ShowsTheRemovalsOfAVaultThatWaitForAnAnswer(t *testing.T) {
	backend, _ := newPausedBackend(t)

	vaults := listVaults(t, backend)

	want := api.PausedDeletions{Count: 2, Tracked: 4, Direction: "send"}
	if len(vaults) != 1 || vaults[0].PausedDeletions == nil || *vaults[0].PausedDeletions != want {
		t.Errorf("vaults = %+v, want the vault with paused deletions %+v", vaults, want)
	}
}

func TestListVaults_ShowsNoPauseForAVaultWhoseRemovalsDoNotWait(t *testing.T) {
	w := newWorld(t)
	w.c.Connect(t, "Test")
	backend := service.NewBackend(w.c.SyncDeps()).WithDeletions(w.c.State, syncwire.NewResolveDeletions(w.c.WireDeps()))

	vaults := listVaults(t, backend)

	if len(vaults) != 1 || vaults[0].PausedDeletions != nil {
		t.Errorf("vaults = %+v, want no paused deletions", vaults)
	}
}

func TestResolveDeletions_AcceptsTheAnswerOfThePersonAndClearsThePause(t *testing.T) {
	backend, w := newPausedBackend(t)

	if _, err := backend.ResolveDeletions(t.Context(), api.ResolveDeletionsRequest{VaultID: "Test", Choice: "keep"}); err != nil {
		t.Fatal(err)
	}

	if vaults := listVaults(t, backend); vaults[0].PausedDeletions != nil {
		t.Errorf("paused deletions = %+v after the answer, want none", vaults[0].PausedDeletions)
	}
	answer, err := w.c.State.ReadApproval(t.Context(), "Test")
	if err != nil || answer == nil || answer.Choice != transfer.ChoiceKeep || answer.Direction != transfer.DirectionSend {
		t.Errorf("answer = %+v (%v), want keep for the send, kept for the next pass", answer, err)
	}
}

func TestResolveDeletions_RefusesAChoiceItDoesNotKnowAndAVaultWithNothingToAnswer(t *testing.T) {
	backend, _ := newPausedBackend(t)

	if _, err := backend.ResolveDeletions(t.Context(), api.ResolveDeletionsRequest{VaultID: "Test", Choice: "later"}); err == nil {
		t.Error("an unknown choice was taken")
	}
	if vaults := listVaults(t, backend); vaults[0].PausedDeletions == nil {
		t.Error("an unknown choice cleared the pause")
	}
	if _, err := backend.ResolveDeletions(t.Context(), api.ResolveDeletionsRequest{VaultID: "Test", Choice: "remove"}); err != nil {
		t.Fatal(err)
	}
	if _, err := backend.ResolveDeletions(t.Context(), api.ResolveDeletionsRequest{VaultID: "Test", Choice: "remove"}); !errors.Is(err, deletions.ErrNothingPending) {
		t.Errorf("an answer with nothing pending = %v, want ErrNothingPending", err)
	}
}
