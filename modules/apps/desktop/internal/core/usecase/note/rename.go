package note

import (
	"context"
	"errors"
	pathpkg "path"
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// Rename gives a note a different name.
//
// A note is shown by its `title`, else by its first level-one heading, else by
// its filename, and whichever of the three names it is brought into line before
// the file is moved. A move refused because the name is taken leaves a note that
// says what it is called under a filename that does not.
type Rename struct {
	Readers port.VaultReaders
	Writers port.VaultWriters
	Links   port.LinkQueries
	Index   func(ctx context.Context, v domain.Vault, paths []string) error
	// Moving is told where the note went, so that whoever is showing it at the
	// name it had follows it. Nothing is told where nobody is drawing.
	Moving Moving
}

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

	// The note says what it is called from here on, so the answer carries the
	// name whatever the file does. It carries the file's new path from the
	// moment the file is at it, and the path the note still has until then.
	res := Renamed{Path: path, Title: title, By: by}
	to := pathpkg.Join(pathpkg.Dir(path), name+pathpkg.Ext(path))
	moved, err := Move{
		Readers: u.Readers, Writers: u.Writers, Links: u.Links,
		Index: u.Index, Moving: u.Moving,
	}.Execute(ctx, v, path, to)
	if moved.Landed {
		res.Path, res.Moved = moved.To, &moved
	}
	return res, err
}
