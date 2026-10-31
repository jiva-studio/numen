package note

import (
	"context"
	pathpkg "path"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"
	"github.com/jiva-studio/numen/libs/notes/ulid"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// Create makes a note.
//
// The title is the whole of the naming: the file is named after it, and a note
// is shown by its `title`, else by its filename. A title the filename cannot
// carry whole is written into the frontmatter, and nothing else writes that key.
type Create struct {
	writers vaultport.VaultWriters
	names   NameQueries
	// index brings the named notes up to date, so a note that is created and then
	// searched for immediately is found.
	index Levels
	// now is when this is happening. An identifier carries it.
	now machineport.Clock
}

// NewCreate builds the use case that creates notes. It takes the vault the note
// is written into, the query for which names are taken, the index update for
// the new file, and the clock.
func NewCreate(
	writers vaultport.VaultWriters, names NameQueries, index Levels, now machineport.Clock,
) Create {
	return Create{writers: writers, names: names, index: index, now: now}
}

// WithIndex replaces the index function.
func (u Create) WithIndex(index Levels) Create {
	u.index = index
	return u
}

// NewNote is what to make.
type NewNote struct {
	Title string
	Body  string
	// Path is the folder it goes in, relative to the root. Empty is the root
	// itself: the application does not arrange anyone's folders.
	Path string
	// Links are what the note is joined to, written together with the note itself,
	// so the note is never created without them.
	Links []domain.Link
}

// CreateResult is the note that now exists.
type CreateResult struct {
	Path  string
	ID    string
	Title string
	// Shares is the other notes already filed under this name. Creating a note
	// under a shared name is allowed and is reported.
	Shares []string
}

func (u Create) Execute(ctx context.Context, v domain.Vault, in NewNote) (CreateResult, error) {
	title := strings.TrimSpace(in.Title)
	name, exact, err := domain.Filename(title)
	if err != nil {
		return CreateResult{}, err
	}
	path := pathpkg.Join(in.Path, name+domain.NoteExtension)

	// Before anything is made: a link the note cannot carry leaves no file.
	for _, link := range in.Links {
		if err := CheckWritable(link); err != nil {
			return CreateResult{}, err
		}
	}

	identifier := ulid.New(u.now.Now())

	content, err := writeTitle(markdown.Create(identifier, in.Body), title, exact)
	if err != nil {
		return CreateResult{}, err
	}
	content, err = writeLinks(content, in.Links)
	if err != nil {
		return CreateResult{}, err
	}

	if err := CheckSize(path, len(content), MaxBytes); err != nil {
		return CreateResult{}, err
	}

	writer, err := u.writers.Open(v)
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
	made := CreateResult{Path: path, ID: identifier, Title: title}
	if err := u.level(ctx, v, path); err != nil {
		return made, err
	}
	shares, err := u.names.GetNamedPaths(ctx, v.ID, domain.Basename(path))
	if err != nil {
		return made, err
	}
	made.Shares = removePath(shares, path)
	return made, nil
}

// writeTitle writes the note's name into its frontmatter, where the filename
// cannot carry the whole of it. A filename that carries it names the note, and
// nothing is written.
func writeTitle(content []byte, title string, exact bool) ([]byte, error) {
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

// writeLinks writes relationships into freshly created frontmatter, through the
// same splicing every other link uses: one set of quoting rules.
func writeLinks(content []byte, links []domain.Link) ([]byte, error) {
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

func (u Create) level(ctx context.Context, v domain.Vault, paths ...string) error {
	if u.index == nil {
		return nil
	}
	return WrapUnlevelled(u.index(ctx, v, paths), paths...)
}

func removePath(paths []string, path string) []string {
	var out []string
	for _, p := range paths {
		if p != path {
			out = append(out, p)
		}
	}
	return out
}
