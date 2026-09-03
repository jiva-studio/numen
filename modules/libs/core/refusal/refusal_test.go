package refusal_test

import (
	"errors"
	"fmt"
	"io/fs"
	"syscall"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/refusal"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// The vocabulary is one. Every window naming a refusal names it from here, so
// this is where the whole of it is written down.

func TestWhichRefusalAnOutcomeIs(t *testing.T) {
	for name, c := range map[string]struct {
		outcome note.ReadOutcome
		want    v1.Refusal
		is      bool
	}{
		"a note that is not there":  {note.Missing, v1.Refusal_REFUSAL_MISSING, true},
		"a file that is not a note": {note.NotANote, v1.Refusal_REFUSAL_NOT_A_NOTE, true},
		"a file that is not text":   {note.NotText, v1.Refusal_REFUSAL_NOT_TEXT, true},
		"a note past its bound":     {note.TooLarge, v1.Refusal_REFUSAL_TOO_LARGE, true},
		"frontmatter nobody can read": {
			note.Unreadable, v1.Refusal_REFUSAL_UNREADABLE, true,
		},
		"a note that was read": {note.Ok, v1.Refusal_REFUSAL_UNSPECIFIED, false},
	} {
		t.Run(name, func(t *testing.T) {
			reason, refused := refusal.Of(c.outcome)
			if refused != c.is || reason != c.want {
				t.Errorf("got %v refused=%v, want %v refused=%v",
					reason, refused, c.want, c.is)
			}
		})
	}
}

func TestWhichRefusalAWritesErrorIs(t *testing.T) {
	for name, c := range map[string]struct {
		err  error
		want v1.Refusal
		is   bool
	}{
		"the vault holds no note there": {
			fmt.Errorf("read Old.md: %w", note.ErrNoNote),
			v1.Refusal_REFUSAL_MISSING, true,
		},
		"more text than a note is written with": {
			fmt.Errorf("%w: 9 bytes", note.ErrTooLarge),
			v1.Refusal_REFUSAL_TOO_LARGE, true,
		},
		"a title no note can be given": {
			fmt.Errorf("%w: nothing to name it", note.ErrUnnameable),
			v1.Refusal_REFUSAL_UNNAMEABLE, true,
		},
		"a frontmatter written on one line": {
			fmt.Errorf("Old.md: %w", note.ErrInline),
			v1.Refusal_REFUSAL_UNREADABLE, true,
		},
		"a frontmatter block that is never closed": {
			fmt.Errorf("Old.md: %w", note.ErrUnterminated),
			v1.Refusal_REFUSAL_UNREADABLE, true,
		},
		"frontmatter that is not YAML": {
			fmt.Errorf("Old.md: %w", note.ErrUnreadable),
			v1.Refusal_REFUSAL_UNREADABLE, true,
		},
		"a whole note handed back as prose": {
			fmt.Errorf("Old.md: %w", note.ErrBodyRefused),
			v1.Refusal_REFUSAL_BODY_REFUSED, true,
		},
		"a file the vault leaves alone": {
			fmt.Errorf("Old.png: %w", port.ErrNotANote),
			v1.Refusal_REFUSAL_NOT_A_NOTE, true,
		},
		"a file where the note would go": {
			fmt.Errorf("Old.md: %w", port.ErrOccupied),
			v1.Refusal_REFUSAL_OCCUPIED, true,
		},
		"a file where a folder of the path must be": {
			fmt.Errorf("a/b/Old.md: %w", syscall.ENOTDIR),
			v1.Refusal_REFUSAL_OCCUPIED, true,
		},
		"the vault itself is out of reach": {
			fmt.Errorf("stat /vault: %w", fs.ErrNotExist),
			v1.Refusal_REFUSAL_UNSPECIFIED, false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			reason, refused := refusal.By(c.err)
			if refused != c.is || reason != c.want {
				t.Errorf("got %v refused=%v, want %v refused=%v",
					reason, refused, c.want, c.is)
			}
		})
	}
}

func TestWhichCodeAnErrorThatIsNoRefusalAnswersWith(t *testing.T) {
	outside := fmt.Errorf("../elsewhere.md: %w", port.ErrOutside)
	if got := refusal.Coded(outside); got != connect.CodeInvalidArgument {
		t.Errorf("a path that leaves the vault answers %v", got)
	}
	if got := refusal.Coded(errors.New("the disk is full")); got != connect.CodeInternal {
		t.Errorf("the vault being out of reach answers %v", got)
	}
}
