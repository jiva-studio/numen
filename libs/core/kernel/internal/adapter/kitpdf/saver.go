package kitpdf

import (
	"bytes"
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	"github.com/jiva-studio/numen/libs/shared/atomicfile"
)

const (
	fileName  = "Numen recovery kit.pdf"
	saveTitle = "Save the recovery kit"
)

// Dialog is the native dialog in which the person chooses the file to save to. It answers false
// when none was chosen.
type Dialog interface {
	ChooseSaveFile(ctx context.Context, title, name string, accept []string) (string, bool, error)
}

// Saver writes the kit as a PDF to the file the person chooses in the dialog. Only the person can
// read the file.
type Saver struct{ dialog Dialog }

// NewSaver returns the saver that asks through dialog. A nil dialog is an installation with no window to show it in.
func NewSaver(dialog Dialog) *Saver { return &Saver{dialog: dialog} }

// SaveKit asks for the file, then writes the kit there. It returns false when no file was chosen,
// and machine.ErrNoFolderDialog when the saver has no dialog.
func (s *Saver) SaveKit(ctx context.Context, kit identitydomain.Kit) (bool, error) {
	if s.dialog == nil {
		return false, machine.ErrNoFolderDialog
	}
	path, isChosen, err := s.dialog.ChooseSaveFile(ctx, saveTitle, fileName, []string{".pdf"})
	if err != nil || !isChosen {
		return false, err
	}
	content, err := Render(kit)
	if err != nil {
		return false, err
	}
	if err := atomicfile.WriteFile(path, bytes.NewReader(content), 0o600); err != nil {
		return false, err
	}
	return true, nil
}
