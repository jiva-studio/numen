package wire_test

import (
	"errors"
	"fmt"
	"io/fs"
	"syscall"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
)

// The vocabulary is one. Every window naming an error names it from here, so
// this is where the whole of it is written down.

func TestWhichErrorCodeAnOutcomeIs(t *testing.T) {
	for name, c := range map[string]struct {
		outcome edit.ReadOutcome
		want    v1.ErrorCode
		is      bool
	}{
		"a note that is not there":  {edit.Missing, v1.ErrorCode_ERROR_CODE_NOT_FOUND, true},
		"a file that is not a note": {edit.NotANote, v1.ErrorCode_ERROR_CODE_NOT_A_NOTE, true},
		"a file that is not text":   {edit.NotText, v1.ErrorCode_ERROR_CODE_NOT_TEXT, true},
		"a note past its bound":     {edit.TooLarge, v1.ErrorCode_ERROR_CODE_TOO_LARGE, true},
		"frontmatter nobody can read": {
			edit.Unreadable, v1.ErrorCode_ERROR_CODE_UNREADABLE, true,
		},
		"a note that was read": {edit.Ok, v1.ErrorCode_ERROR_CODE_UNSPECIFIED, false},
	} {
		t.Run(name, func(t *testing.T) {
			reason, ok := wire.ConvertOutcomeToErrorCode(c.outcome)
			if ok != c.is || reason != c.want {
				t.Errorf("got %v ok=%v, want %v ok=%v",
					reason, ok, c.want, c.is)
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
			fmt.Errorf("read Old.md: %w", edit.ErrNoNote),
			v1.ErrorCode_ERROR_CODE_NOT_FOUND, true,
		},
		"more text than a note is written with": {
			fmt.Errorf("%w: 9 bytes", edit.ErrTooLarge),
			v1.ErrorCode_ERROR_CODE_TOO_LARGE, true,
		},
		"a title no note can be given": {
			fmt.Errorf("%w: nothing to name it", domain.ErrInvalidName),
			v1.ErrorCode_ERROR_CODE_INVALID_NAME, true,
		},
		"a frontmatter written on one line": {
			fmt.Errorf("Old.md: %w", edit.ErrInline),
			v1.ErrorCode_ERROR_CODE_UNREADABLE, true,
		},
		"a frontmatter block that is never closed": {
			fmt.Errorf("Old.md: %w", edit.ErrUnterminated),
			v1.ErrorCode_ERROR_CODE_UNREADABLE, true,
		},
		"a frontmatter carrying a YAML anchor": {
			fmt.Errorf("Old.md: %w", edit.ErrAnchored),
			v1.ErrorCode_ERROR_CODE_UNREADABLE, true,
		},
		"frontmatter that is not YAML": {
			fmt.Errorf("Old.md: %w", edit.ErrUnreadable),
			v1.ErrorCode_ERROR_CODE_UNREADABLE, true,
		},
		"a whole note returned as prose": {
			fmt.Errorf("Old.md: %w", edit.ErrBodyUnwritable),
			v1.ErrorCode_ERROR_CODE_BODY_UNWRITABLE, true,
		},
		"a file the vault leaves alone": {
			fmt.Errorf("Old.png: %w", edit.ErrNotANote),
			v1.ErrorCode_ERROR_CODE_NOT_A_NOTE, true,
		},
		"a file where the note would go": {
			fmt.Errorf("Old.md: %w", vault.ErrAlreadyExists),
			v1.ErrorCode_ERROR_CODE_ALREADY_EXISTS, true,
		},
		"a file where a folder of the path must be": {
			fmt.Errorf("a/b/Old.md: %w", syscall.ENOTDIR),
			v1.ErrorCode_ERROR_CODE_ALREADY_EXISTS, true,
		},
		"the vault itself is out of reach": {
			fmt.Errorf("stat /vault: %w", fs.ErrNotExist),
			v1.ErrorCode_ERROR_CODE_UNSPECIFIED, false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			reason, ok := wire.GetErrorCode(c.err)
			if ok != c.is || reason != c.want {
				t.Errorf("got %v ok=%v, want %v ok=%v",
					reason, ok, c.want, c.is)
			}
		})
	}
}

func TestWhichCodeAnErrorTheSchemaDoesNotCarryAnswersWith(t *testing.T) {
	outside := fmt.Errorf("../elsewhere.md: %w", vault.ErrOutside)
	if got := wire.GetCode(outside); got != connect.CodeInvalidArgument {
		t.Errorf("a path that leaves the vault answers %v", got)
	}
	if got := wire.GetCode(errors.New("the disk is full")); got != connect.CodeInternal {
		t.Errorf("the vault being out of reach answers %v", got)
	}
}
