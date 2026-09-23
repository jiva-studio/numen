package wire_test

import (
	"errors"
	"fmt"
	"io/fs"
	"syscall"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// The vocabulary is one. Every window naming an error names it from here, so
// this is where the whole of it is written down.

func TestWhichErrorCodeAnOutcomeIs(t *testing.T) {
	for name, c := range map[string]struct {
		outcome note.ReadOutcome
		want    v1.ErrorCode
		is      bool
	}{
		"a note that is not there":  {note.Missing, v1.ErrorCode_ERROR_CODE_MISSING, true},
		"a file that is not a note": {note.NotANote, v1.ErrorCode_ERROR_CODE_NOT_A_NOTE, true},
		"a file that is not text":   {note.NotText, v1.ErrorCode_ERROR_CODE_NOT_TEXT, true},
		"a note past its bound":     {note.TooLarge, v1.ErrorCode_ERROR_CODE_TOO_LARGE, true},
		"frontmatter nobody can read": {
			note.Unreadable, v1.ErrorCode_ERROR_CODE_UNREADABLE, true,
		},
		"a note that was read": {note.Ok, v1.ErrorCode_ERROR_CODE_UNSPECIFIED, false},
	} {
		t.Run(name, func(t *testing.T) {
			reason, refused := wire.ErrorCodeOf(c.outcome)
			if refused != c.is || reason != c.want {
				t.Errorf("got %v refused=%v, want %v refused=%v",
					reason, refused, c.want, c.is)
			}
		})
	}
}

func TestWhichErrorCodeAWritesErrorIs(t *testing.T) {
	for name, c := range map[string]struct {
		err  error
		want v1.ErrorCode
		is   bool
	}{
		"the vault holds no note there": {
			fmt.Errorf("read Old.md: %w", note.ErrNoNote),
			v1.ErrorCode_ERROR_CODE_MISSING, true,
		},
		"more text than a note is written with": {
			fmt.Errorf("%w: 9 bytes", note.ErrTooLarge),
			v1.ErrorCode_ERROR_CODE_TOO_LARGE, true,
		},
		"a title no note can be given": {
			fmt.Errorf("%w: nothing to name it", note.ErrUnnameable),
			v1.ErrorCode_ERROR_CODE_UNNAMEABLE, true,
		},
		"a frontmatter written on one line": {
			fmt.Errorf("Old.md: %w", note.ErrInline),
			v1.ErrorCode_ERROR_CODE_UNREADABLE, true,
		},
		"a frontmatter block that is never closed": {
			fmt.Errorf("Old.md: %w", note.ErrUnterminated),
			v1.ErrorCode_ERROR_CODE_UNREADABLE, true,
		},
		"a frontmatter carrying a YAML anchor": {
			fmt.Errorf("Old.md: %w", note.ErrAnchored),
			v1.ErrorCode_ERROR_CODE_UNREADABLE, true,
		},
		"frontmatter that is not YAML": {
			fmt.Errorf("Old.md: %w", note.ErrUnreadable),
			v1.ErrorCode_ERROR_CODE_UNREADABLE, true,
		},
		"a whole note handed back as prose": {
			fmt.Errorf("Old.md: %w", note.ErrBodyUnwritable),
			v1.ErrorCode_ERROR_CODE_BODY_UNWRITABLE, true,
		},
		"a file the vault leaves alone": {
			fmt.Errorf("Old.png: %w", port.ErrNotANote),
			v1.ErrorCode_ERROR_CODE_NOT_A_NOTE, true,
		},
		"a file where the note would go": {
			fmt.Errorf("Old.md: %w", port.ErrOccupied),
			v1.ErrorCode_ERROR_CODE_OCCUPIED, true,
		},
		"a file where a folder of the path must be": {
			fmt.Errorf("a/b/Old.md: %w", syscall.ENOTDIR),
			v1.ErrorCode_ERROR_CODE_OCCUPIED, true,
		},
		"the vault itself is out of reach": {
			fmt.Errorf("stat /vault: %w", fs.ErrNotExist),
			v1.ErrorCode_ERROR_CODE_UNSPECIFIED, false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			reason, refused := wire.ErrorCodeBy(c.err)
			if refused != c.is || reason != c.want {
				t.Errorf("got %v refused=%v, want %v refused=%v",
					reason, refused, c.want, c.is)
			}
		})
	}
}

func TestWhichCodeAnErrorTheSchemaDoesNotCarryAnswersWith(t *testing.T) {
	outside := fmt.Errorf("../elsewhere.md: %w", port.ErrOutside)
	if got := wire.GetCode(outside); got != connect.CodeInvalidArgument {
		t.Errorf("a path that leaves the vault answers %v", got)
	}
	if got := wire.GetCode(errors.New("the disk is full")); got != connect.CodeInternal {
		t.Errorf("the vault being out of reach answers %v", got)
	}
}
