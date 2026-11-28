package pull

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
)

func TestCheckRoom_RefusesAListAtTheLimit(t *testing.T) {
	if err := checkRoom(syncprotocol.MaxCatalogEntries - 1); err != nil {
		t.Errorf("a list one short of the limit = %v, want room", err)
	}
	if err := checkRoom(syncprotocol.MaxCatalogEntries); !errors.Is(err, syncprotocol.ErrTooManyPages) {
		t.Errorf("a list at the limit = %v, want ErrTooManyPages", err)
	}
}

func TestQueueRemoval_FailsWhenTheQueueIsAtTheLimit(t *testing.T) {
	r := &run{removals: make([]removal, syncprotocol.MaxCatalogEntries)}

	if err := r.queueRemoval(removal{}, 1); !errors.Is(err, syncprotocol.ErrTooManyPages) {
		t.Errorf("queue at the limit = %v, want ErrTooManyPages", err)
	}
}

func TestRecordRollback_FailsWhenTheListIsAtTheLimit(t *testing.T) {
	r := &run{rolledBack: make([]string, syncprotocol.MaxCatalogEntries)}

	if err := r.recordRollback(note.Meta{}, &syncv1.Revision{}); !errors.Is(err, syncprotocol.ErrTooManyPages) {
		t.Errorf("list at the limit = %v, want ErrTooManyPages", err)
	}
}
