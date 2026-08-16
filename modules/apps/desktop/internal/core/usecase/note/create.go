package note

import (
	"context"
	"errors"
	pathpkg "path"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/ulid"
)

// Create makes a note.
//
// The title is the whole of the naming: the file is named after it, and a note
// is shown by its title, else by its first heading, else by its filename.
// Nothing writes a `title` key, because nothing needs to.
type Create struct {
	Writers port.VaultWriters
	Names   port.NoteQueries
	// Index brings the named notes up to date, so that a caller which creates
	// a note and searches for it in the next breath finds it.
	Index func(ctx context.Context, v domain.Vault, paths []string) error
	// Extension is what a new note is filed under. Empty means markdown.
	Extension string
	// Now is when this is happening. An identifier carries it.
	Now func() time.Time
}

// NewNote is what to make.
type NewNote struct {
	Title string
	Body  string
	// Folder is where in the vault it goes, relative to the root. Empty is the
	// root itself: the application does not arrange anyone's folders.
	Folder string
}

// Created is the note that now exists.
type Created struct {
	Path       string
	Identifier string
	Title      string
	// Shares is the other notes already filed under this name. Creating one
	// anyway is allowed and said out loud, because refusing would be an
	// invariant the application cannot hold.
	Shares []string
}

func (u Create) Execute(ctx context.Context, v domain.Vault, in NewNote) (Created, error) {
	name, exact := domain.Filename(in.Title)
	if name == "" {
		return Created{}, errors.New("a note needs a title that can be a filename")
	}
	path := pathpkg.Join(in.Folder, name+u.extension())

	identifier, err := ulid.New(u.now())
	if err != nil {
		return Created{}, err
	}

	body := in.Body
	if !exact {
		// The name could not carry the title, so the body does. The order a
		// note is named by finds it either way.
		body = "# " + in.Title + "\n\n" + body
	}

	writer, err := u.Writers.Open(v)
	if err != nil {
		return Created{}, err
	}
	// Create rather than write: whether the path was free is the filesystem's
	// to answer, once, rather than something asked beforehand and hoped to
	// still be true.
	if err := writer.Create(ctx, path, markdown.Create(identifier, body)); err != nil {
		return Created{}, err
	}
	if err := u.index(ctx, v, path); err != nil {
		return Created{}, err
	}

	shares, err := u.Names.Named(ctx, v.ID, domain.Basename(path))
	if err != nil {
		return Created{}, err
	}
	return Created{
		Path:       path,
		Identifier: identifier,
		Title:      in.Title,
		Shares:     without(shares, path),
	}, nil
}

func (u Create) extension() string {
	if u.Extension == "" {
		return ".md"
	}
	return u.Extension
}

func (u Create) now() time.Time {
	if u.Now == nil {
		return time.Now()
	}
	return u.Now()
}

func (u Create) index(ctx context.Context, v domain.Vault, paths ...string) error {
	if u.Index == nil {
		return nil
	}
	return u.Index(ctx, v, paths)
}

func without(paths []string, path string) []string {
	var out []string
	for _, p := range paths {
		if p != path {
			out = append(out, p)
		}
	}
	return out
}
