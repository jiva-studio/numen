package note

import (
	"context"
	"errors"
	pathpkg "path"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
)

// Rename gives a note a different name.
//
// A note is shown by its `title`, else by its filename, and whichever of the
// two names it is brought into line first. The file follows where a title and a
// filename are kept as one name, and a move refused because the name is taken
// leaves a note that says what it is called under a filename that does not.
type Rename struct{ Move }

// RenameResult says what the note is called now and what the file did.
type RenameResult struct {
	Path  string // where the note is filed now
	Title string
	By    NamedBy
	Moved *MoveResult // nil when the file is not at a different path
}

// errFilenameNamesIt ends the edit without writing. A note its filename names
// has nothing in it to bring into line, and is moved and not edited.
var errFilenameNamesIt = errors.New("the filename says it")

func (u Rename) Execute(ctx context.Context, v domain.Vault, path, title string) (RenameResult, error) {
	title = strings.TrimSpace(title)
	name, exact, err := domain.Filed(title)
	if err != nil {
		return RenameResult{}, err
	}

	var by NamedBy
	e := Editing{Readers: u.Readers, Writers: u.Writers, Index: u.Index}
	_, err = e.Apply(ctx, v, path, func(doc *markdown.Document) error {
		if _, titled := doc.Title(); titled {
			by = ByFrontmatter
			return doc.SetTitle(title)
		}
		if exact {
			by = ByFilename
			return errFilenameNamesIt
		}
		// The filename cannot carry the title, so the frontmatter takes it.
		by = ByFrontmatter
		return doc.SetTitle(title)
	})
	if err != nil && !errors.Is(err, errFilenameNamesIt) {
		return RenameResult{}, err
	}

	// The answer carries the note's name whatever the file does, the file's new
	// path once the file is at it, and the path it still has until then.
	res := RenameResult{Path: path, Title: title, By: by}
	if moves, _ := u.Sync.Kept().Renaming(by); !moves {
		return res, nil
	}
	to := pathpkg.Join(pathpkg.Dir(path), name+pathpkg.Ext(path))
	moved, err := u.Move.Execute(ctx, v, path, to)
	if moved.Landed {
		res.Path, res.Moved = moved.To, &moved
	}
	return res, err
}

// Called writes into the note at this path the name its file carries, where a
// title and a filename are kept as one name. A note its filename names carries
// its name nowhere else, and nothing is added to it.
func (u Move) Called(ctx context.Context, v domain.Vault, path string) error {
	// The note is opened only where the setting writes into it.
	if _, writes := u.Sync.Kept().Renaming(ByFrontmatter); !writes {
		return nil
	}

	name := domain.Basename(path)
	e := Editing{Readers: u.Readers, Writers: u.Writers, Index: u.Index}
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
