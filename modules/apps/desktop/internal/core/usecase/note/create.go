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
	// Links are what it is joined to, written in the same breath as the note
	// itself, so that it never exists as an island.
	Links []domain.Link
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

	// Before anything is made: a link the note cannot carry leaves no file.
	for _, link := range in.Links {
		if err := Writable(link); err != nil {
			return Created{}, err
		}
	}

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

	content, err := joined(markdown.Create(identifier, body), in.Links)
	if err != nil {
		return Created{}, err
	}

	writer, err := u.Writers.Open(v)
	if err != nil {
		return Created{}, err
	}
	// Create rather than write: whether the path was free is the filesystem's
	// to answer, once, rather than something asked beforehand and hoped to
	// still be true.
	if err := writer.Create(ctx, path, content); err != nil {
		return Created{}, err
	}

	// The note is on disk from here on, so everything after it answers with
	// where it is, whether or not it succeeds.
	made := Created{Path: path, Identifier: identifier, Title: in.Title}
	if err := u.index(ctx, v, path); err != nil {
		return made, err
	}
	shares, err := u.Names.Named(ctx, v.ID, domain.Basename(path))
	if err != nil {
		return made, err
	}
	made.Shares = without(shares, path)
	return made, nil
}

// joined writes relationships into frontmatter that has just been made, through
// the splicing every other link goes through: one set of quoting rules, not two.
func joined(content []byte, links []domain.Link) ([]byte, error) {
	if len(links) == 0 {
		return content, nil
	}
	doc, err := markdown.Open(content)
	if err != nil {
		return nil, err
	}
	for _, link := range links {
		if err := doc.AddLink(link); err != nil {
			return nil, err
		}
	}
	return doc.Bytes(), nil
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
