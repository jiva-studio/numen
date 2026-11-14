package note

import (
	"context"
	"errors"
	"fmt"
	pathpkg "path"
	"strings"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/markdown"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/naming"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
	"github.com/jiva-studio/numen/libs/note/edit"
)

// Rename gives a note a different name.
//
// A note is shown by its `title`, else by its filename, and whichever of the
// two names it is updated first. The file follows when title and filename are
// kept as one name, so the rename does not leave a title different from the filename.
type Rename struct {
	readers vaultport.VaultReaders
	writers MoveWriting
	index   edit.IndexUpdater
	now     machineport.Clock
	sync    naming.SyncSetting
	moving  fileMoving
	gate    windowport.WriteGate
}

// fileMoving files a note at another path. A note that traveled because it was
// renamed and one that traveled because it was filed elsewhere settle the same
// way, so both go through the one mover.
type fileMoving interface {
	Execute(ctx context.Context, v domain.Vault, from, to string) (domain.MoveResult, error)
}

// NewRename is the rename over the mover that files the note. It takes the readers,
// writers, index updater, clock and naming setting of d, and uses the rest of d
// nowhere. A rename passes gate.
func NewRename(d MoveDeps, moving fileMoving, gate windowport.WriteGate) Rename {
	return Rename{readers: d.Readers, writers: d.Writers, index: d.Index, now: d.Now, sync: d.Sync, moving: moving, gate: gate}
}

// RenameResult reports the note's current name and what happened to the file.
type RenameResult struct {
	Path  string
	Title string
	By    naming.Source
	Moved *domain.MoveResult
}

// errFilenameNamesIt ends the edit without writing. A note its filename names
// has nothing in it to bring into line, and is moved and not edited.
var errFilenameNamesIt = errors.New("the filename determines the title")

// Execute renames the note at path to title. It fails with window.ErrClosing once the window is closing.
func (u Rename) Execute(ctx context.Context, v domain.Vault, path, title string) (RenameResult, error) {
	return writeUnder(u.gate, func() (RenameResult, error) { return u.rename(ctx, v, path, title) })
}

func (u Rename) rename(ctx context.Context, v domain.Vault, path, title string) (RenameResult, error) {
	title = strings.TrimSpace(title)
	name, exact, err := domain.Filename(title)
	if err != nil {
		return RenameResult{}, err
	}

	to := pathpkg.Join(pathpkg.Dir(path), name+pathpkg.Ext(path))
	var source naming.Source
	e := NewEdit(u.readers, u.writers, u.index, u.now)
	_, err = e.Apply(ctx, v, path, func(doc *markdown.Document) error {
		if _, titled := doc.Title(); titled || !exact {
			// A title key keeps the name, and so does a title that cannot be stored in a filename.
			source = naming.ByFrontmatter
			if err := u.checkTarget(ctx, v, path, to, source); err != nil {
				return err
			}
			return doc.SetTitle(title)
		}
		source = naming.ByFilename
		return errFilenameNamesIt
	})
	if errors.Is(err, vault.ErrAlreadyExists) {
		return RenameResult{Path: path, Title: title, By: source},
			domain.NewDetailError(fmt.Sprintf("%s already exists; %s is unchanged", to, path), err)
	}
	if err != nil && !errors.Is(err, errFilenameNamesIt) {
		return RenameResult{}, err
	}

	// The answer carries the note's name whatever the file does, the file's new
	// path once the file is at it, and the path it still has until then.
	res := RenameResult{Path: path, Title: title, By: source}
	if moves, _ := u.sync.GetSetting().GetRenameEffects(source); !moves {
		return res, nil
	}
	moved, err := u.moving.Execute(ctx, v, path, to)
	if moved.IsMoved {
		res.Path, res.Moved = moved.To, &moved
	}
	if source == naming.ByFilename && errors.Is(err, vault.ErrAlreadyExists) {
		// A note whose title is stored in its filename was not written before the move, so it is unchanged.
		return res, domain.NewDetailError(fmt.Sprintf("%s already exists; %s is unchanged", to, path), err)
	}
	return res, err
}

func (u Rename) checkTarget(ctx context.Context, v domain.Vault, from, to string, source naming.Source) error {
	if moves, _ := u.sync.GetSetting().GetRenameEffects(source); !moves {
		return nil
	}
	mover, err := u.writers.OpenFileMover(v)
	if err != nil {
		return err
	}
	return mover.CheckMoveTarget(ctx, from, to)
}

// WriteFilenameAsTitle writes into the note at this path the name its file
// carries, where a title and a filename are kept as one name. A note its
// filename names carries its name nowhere else, and nothing is added to it.
func (u Move) WriteFilenameAsTitle(ctx context.Context, v domain.Vault, path string) error {
	// The note is opened only where the setting writes into it.
	if _, writes := u.sync.GetSetting().GetRenameEffects(naming.ByFrontmatter); !writes {
		return nil
	}

	name := domain.Basename(path)
	e := NewEdit(u.readers, u.writers, u.index, u.now)
	_, err := e.Apply(ctx, v, path, func(doc *markdown.Document) error {
		if _, titled := doc.Title(); titled {
			return doc.SetTitle(name)
		}
		return errFilenameNamesIt
	})
	switch {
	case errors.Is(err, errFilenameNamesIt):
		return nil
	case errors.Is(err, markdown.ErrUnreadable),
		errors.Is(err, markdown.ErrInline),
		errors.Is(err, markdown.ErrUnterminated),
		errors.Is(err, markdown.ErrAnchored):
		// A note whose frontmatter cannot be read is never written, and its
		// file is renamed like any other. These four are the whole of what a
		// frontmatter that cannot be changed a key at a time answers with.
		return nil
	}
	return err
}
