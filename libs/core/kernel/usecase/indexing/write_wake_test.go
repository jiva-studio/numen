package indexing

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"
)

func TestCommittedWritesWakeOnlyThePassOverTheirVault(t *testing.T) {
	worker := NewWorker(Config{})
	writes := WriteWake(worker.WakeWrittenVault)
	first := domain.Vault{ID: "first"}
	second := domain.Vault{ID: "second"}
	session := worker.NewSession(first)
	writes.ReportWrites(first.ID, nil)
	prior := worker.newPass(session, false)
	current := worker.newPass(worker.NewSession(second), false)
	writes.ReportWrites(first.ID, nil)
	writes.ReportWrites(second.ID)
	writes.ReportWrites(second.ID, errors.New("stale baseline"))

	if !prior.waits.hasWakes() || current.waits.hasWakes() {
		t.Fatal("the new vault inherited a write from the previous vault or a rejected write")
	}
	writes.ReportWrites(second.ID, errors.New("refused"), fmt.Errorf("refresh: %w", edit.ErrIndexOutdated))
	if !current.waits.hasWakes() {
		t.Fatal("a partial batch with an outdated main index left no pending vector pass")
	}
}
