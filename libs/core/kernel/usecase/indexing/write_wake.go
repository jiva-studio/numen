package indexing

import (
	"errors"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"
)

// WriteWake schedules a vector pass for the vault whose files changed.
// A caller without a vector worker leaves it nil.
type WriteWake func(domain.VaultID)

// ReportWrites schedules one pass if any write committed, including a committed
// file whose main index update failed. An empty batch schedules no pass.
func (w WriteWake) ReportWrites(v domain.VaultID, outcomes ...error) {
	if w == nil {
		return
	}
	for _, err := range outcomes {
		if err == nil || errors.Is(err, edit.ErrIndexOutdated) {
			w(v)
			return
		}
	}
}

// WakeWrittenVault schedules the pass currently indexing id. A pass over another
// vault processes its own pending vectors when it opens.
func (w *Worker) WakeWrittenVault(id domain.VaultID) {
	signals := w.wakes.Load()
	if signals.vault == id {
		signals.wakeNotes()
	}
}
