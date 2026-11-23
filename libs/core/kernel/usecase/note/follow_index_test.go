package note_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/shared/note/edit"
)

func TestFollowIndex_DropsOnlyTheOutdatedIndexError(t *testing.T) {
	t.Parallel()
	refused := errors.New("refused")
	for name, c := range map[string]struct {
		err      error
		outdated bool
		want     error
	}{
		"written":           {},
		"index not updated": {err: fmt.Errorf("%w: a.md", edit.ErrIndexOutdated), outdated: true},
		"other failure":     {err: refused, want: refused},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			outdated, err := note.NewFollowIndex(&followReport{}).Settle(c.err)
			if outdated != c.outdated || !errors.Is(err, c.want) || (c.want == nil && err != nil) {
				t.Errorf("got outdated %v, err %v; want %v, %v", outdated, err, c.outdated, c.want)
			}
		})
	}
}
