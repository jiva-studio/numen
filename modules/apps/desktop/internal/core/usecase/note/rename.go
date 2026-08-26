package note

import (
	"context"
	"errors"
	pathpkg "path"
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
)

// Rename gives a note a different name.
//
// A note is shown by its `title`, else by its first level-one heading, else by
// its filename, and whichever of the three names it is brought into line first.
// The file follows where a title and a filename are kept as one name, and a move
// refused because the name is taken leaves a note that says what it is called
// under a filename that does not.
type Rename struct{ Move }

// Renamed says what the note is called now and what the file did.
type Renamed struct {
	Path  string // where the note is filed now
	Title string
	By    Naming
	Moved *Moved // nil when the file is not at a different path
}

// errFilenameNamesIt ends the edit without writing. A note its filename names
// has nothing in it to bring into line, and is moved and not edited.
var errFilenameNamesIt = errors.New("the filename says it")

func (u Rename) Execute(ctx context.Context, v domain.Vault, path, title string) (Renamed, error) {
	title = strings.TrimSpace(title)
	name, exact, err := nameOf(title)
	if err != nil {
		return Renamed{}, err
	}

	var by Naming
	e := editing{readers: u.Readers, writers: u.Writers, index: u.Index}
	_, err = e.apply(ctx, v, path, func(doc *markdown.Document) error {
		if _, titled := doc.Title(); titled {
			by = ByFrontmatter
			return doc.SetTitle(title)
		}
		switch written, err := doc.SetHeading(title); {
		case err != nil:
			return err
		case written:
			by = ByHeading
			return nil
		}
		if exact {
			by = ByFilename
			return errFilenameNamesIt
		}
		// The name cannot carry the title, so the body does. The order a note is
		// named by finds it either way.
		by = ByHeading
		return doc.InsertHeading(title)
	})
	if err != nil && !errors.Is(err, errFilenameNamesIt) {
		return Renamed{}, err
	}

	// The answer carries the note's name whatever the file does, the file's new
	// path once the file is at it, and the path it still has until then.
	res := Renamed{Path: path, Title: title, By: by}
	if moves, _ := u.Sync.Renaming(by); !moves {
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
	name := domain.Basename(path)
	e := editing{readers: u.Readers, writers: u.Writers, index: u.Index}
	_, err := e.apply(ctx, v, path, func(doc *markdown.Document) error {
		if _, titled := doc.Title(); titled {
			if _, writes := u.Sync.Renaming(ByFrontmatter); !writes {
				return errFilenameNamesIt
			}
			return doc.SetTitle(name)
		}
		if _, writes := u.Sync.Renaming(ByHeading); !writes {
			return errFilenameNamesIt
		}
		switch written, err := doc.SetHeading(name); {
		case err != nil:
			return err
		case written:
			return nil
		}
		return errFilenameNamesIt
	})
	if errors.Is(err, errFilenameNamesIt) {
		return nil
	}
	return err
}
