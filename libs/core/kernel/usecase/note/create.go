package note

import (
	"context"
	"crypto/rand"
	pathpkg "path"
	"strings"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
	"github.com/jiva-studio/numen/libs/note/markdown"
	"github.com/jiva-studio/numen/libs/note/ulid"

	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Create makes a note.
//
// The title is the whole of the naming: the file is named after it, and a note
// is shown by its `title`, else by its filename. A title the filename cannot
// carry whole is written into the frontmatter, and nothing else writes that key.
type Create struct {
	writers vaultport.NoteWriters
	names   edit.NameQueries
	index   edit.IndexUpdater
	now     machineport.Clock
}

// NewCreate builds the use case that creates notes. It takes the vault the note
// is written into, the query for which names are taken, the index update for
// the new file, and the clock.
func NewCreate(
	writers vaultport.NoteWriters, names edit.NameQueries, index edit.IndexUpdater, now machineport.Clock,
) Create {
	return Create{writers: writers, names: names, index: index, now: now}
}

// NewNote is what to make. CreateNote resolves Named into Links.
type NewNote struct {
	Title string
	Body  string
	Path  string
	Links []domain.Link
	Named []NamedLink
}

// CreateResult is the note that now exists.
type CreateResult struct {
	Path   string
	ID     string
	Title  string
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

	identifier, err := ulid.NewGenerator(rand.Reader).NewID(u.now.Now())
	if err != nil {
		return CreateResult{}, err
	}

	content, err := writeTitle(markdown.Create(identifier, in.Body), title, exact)
	if err != nil {
		return CreateResult{}, err
	}
	content, err = writeLinks(content, in.Links)
	if err != nil {
		return CreateResult{}, err
	}

	if err := edit.CheckSize(path, len(content), edit.MaxBytes); err != nil {
		return CreateResult{}, err
	}

	writer, err := u.writers.OpenNoteWriter(v)
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
	if err := u.updateIndex(ctx, v, path); err != nil {
		return made, err
	}
	shares, err := u.names.GetNamedPaths(ctx, v.ID, domain.Basename(path))
	if err != nil {
		return made, err
	}
	made.Shares = removePath(shares, path)
	return made, nil
}

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

func (u Create) updateIndex(ctx context.Context, v domain.Vault, paths ...string) error {
	if u.index == nil {
		return nil
	}
	// The files are written, so the index follows them even when the caller is canceled.
	return edit.WrapIndexOutdated(u.index(context.WithoutCancel(ctx), v, paths), paths...)
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
