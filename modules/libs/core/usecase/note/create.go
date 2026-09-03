package note

import (
	"context"
	pathpkg "path"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ulid"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Create makes a note.
//
// The title is the whole of the naming: the file is named after it, and a note
// is shown by its `title`, else by its filename. A title the filename cannot
// carry whole is written into the frontmatter, and nothing else writes that key.
type Create struct {
	Writers port.VaultWriters
	Names   port.NoteQueries
	// Index brings the named notes up to date, so that a caller which creates
	// a note and searches for it in the next breath finds it.
	Index func(ctx context.Context, v domain.Vault, paths []string) error
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

// CreateResult is the note that now exists.
type CreateResult struct {
	Path       string
	Identifier string
	Title      string
	// Shares is the other notes already filed under this name. Creating one
	// anyway is allowed, and said out loud.
	Shares []string
}

func (u Create) Execute(ctx context.Context, v domain.Vault, in NewNote) (CreateResult, error) {
	title := strings.TrimSpace(in.Title)
	name, exact, err := domain.Filed(title)
	if err != nil {
		return CreateResult{}, err
	}
	path := pathpkg.Join(in.Folder, name+domain.NoteExtension)

	// Before anything is made: a link the note cannot carry leaves no file.
	for _, link := range in.Links {
		if err := Writable(link); err != nil {
			return CreateResult{}, err
		}
	}

	identifier, err := ulid.New(u.now())
	if err != nil {
		return CreateResult{}, err
	}

	content, err := titled(markdown.Create(identifier, in.Body), title, exact)
	if err != nil {
		return CreateResult{}, err
	}
	content, err = joined(content, in.Links)
	if err != nil {
		return CreateResult{}, err
	}

	if err := Bounded(path, len(content), MaxBytes); err != nil {
		return CreateResult{}, err
	}

	writer, err := u.Writers.Open(v)
	if err != nil {
		return CreateResult{}, err
	}
	// Whether the path was free is the filesystem's to answer, at the moment
	// the file is made.
	if err := writer.Create(ctx, path, content); err != nil {
		return CreateResult{}, err
	}

	// The note is on disk from here on, so everything after it answers with
	// where it is, whether or not it succeeds.
	made := CreateResult{Path: path, Identifier: identifier, Title: title}
	if err := u.index(ctx, v, path); err != nil {
		return made, err
	}
	shares, err := u.Names.Named(ctx, string(v.ID), domain.Basename(path))
	if err != nil {
		return made, err
	}
	made.Shares = without(shares, path)
	return made, nil
}

// titled writes the note's name into its frontmatter, where the filename
// cannot carry the whole of it. A filename that carries it names the note, and
// nothing is written.
func titled(content []byte, title string, exact bool) ([]byte, error) {
	if exact {
		return content, nil
	}
	doc, err := markdown.Open(content)
	if err != nil {
		return nil, err
	}
	if err := doc.SetTitle(title); err != nil {
		return nil, err
	}
	return doc.Bytes(), nil
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
