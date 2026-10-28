package edit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"
	"github.com/jiva-studio/numen/libs/notes/ulid"
)

// Files is the vault's files as an edit works them.
type Files interface {
	Reader
	// Write replaces the file if it is still what the file stat says, else it returns ErrStale.
	// An empty file stat holds the write to nothing.
	Write(ctx context.Context, path string, content []byte, stat domain.FileStat) (domain.FileStat, error)
	// Lock holds the vault's writers until the release is called.
	Lock(ctx context.Context) (release func(), err error)
}

// Levels brings the named notes up to date in the index, so that what a write changed is findable
// before the write is reported done.
type Levels func(ctx context.Context, v domain.Vault, paths []string) error

// Baseline is what a caller last saw of the note it is saving.
//
// Prose answers first: text that is still what the caller was given is the text it read, whatever
// the file's size and time say. A synchroniser, a checkout and a touch all move those over text that
// did not change, and comparing them alone is what asks a person about a file nobody edited.
//
// FileStat answers for text that did move, and is what makes a save that follows a save land: the
// note is at the file stat the last write produced, so it is the note this caller put there.
type Baseline struct {
	// Prose is what a read gave this caller, with every line break as one \n.
	Prose string
	// FileStat is the file that read came out of.
	FileStat domain.FileStat
}

// isStale reports whether the note in front of the writer holds prose this caller has not read.
func (s *Baseline) isStale(on domain.FileStat, prose string) bool {
	if s == nil {
		return false
	}
	return prose != s.Prose && !on.IsUnchanged(s.FileStat)
}

// Edit is what every change to the contents of an existing note needs.
//
// It is one function because the shape is always the same and the rules in it are easy to forget one
// at a time: read, refuse what cannot be read, write the identifier because this is an edit, change
// the one thing, put it back, and bring the index level.
type Edit struct {
	// Open gives the files of the vault.
	Open func(v domain.Vault) (Files, error)
	// Index brings the index level with what was written. Nil brings nothing.
	Index Levels
	// Now is the time an identifier is minted at.
	Now func() time.Time
	// Entropy is the randomness source for minted identifiers.
	Entropy io.Reader

	FileStat domain.FileStat
	// ShouldOverwrite is a caller writing what is in front of the person: the note on disk is replaced
	// stamped.
	ShouldOverwrite bool
	// Bound is the most the file may be. Zero holds it to nothing.
	Bound int
	// Baseline is what the caller last saw of the note, and is what a file that is there is compared
	// with. Nil for a caller that puts its text down whatever the note now holds.
	Baseline *Baseline
}

// Apply makes one change to the note at path and puts it back.
func (e Edit) Apply(ctx context.Context, v domain.Vault, path string, change func(*markdown.Document) error) (domain.FileStat, error) {
	content, err := e.splice(ctx, v, path, change)
	if err != nil {
		return domain.FileStat{}, err
	}
	// The file is on disk, so the file stat is stored alongside whatever the levelling produced and a
	// caller can tell the two apart.
	if e.Index == nil {
		return content, nil
	}
	return content, WrapUnlevelled(e.Index(ctx, v, []string{path}), path)
}

// splice is the read, the change and the write, under this vault's write lock from before the read
// until after the file is replaced.
func (e Edit) splice(
	ctx context.Context, v domain.Vault, path string, change func(*markdown.Document) error,
) (domain.FileStat, error) {
	files, err := e.Open(v)
	if err != nil {
		return domain.FileStat{}, err
	}
	release, err := files.Lock(ctx)
	if err != nil {
		return domain.FileStat{}, err
	}
	defer release()

	// What the file is, asked before its bytes are read, which is the order a read hands its
	// file stat over in.
	on, looked := e.statNote(ctx, files, path)
	raw, err := e.readNote(ctx, files, path)
	if err != nil {
		return domain.FileStat{}, err
	}
	against, err := e.getExpected(path, on, looked)
	if err != nil {
		return domain.FileStat{}, err
	}
	doc, err := markdown.Open(raw)
	if err != nil {
		return domain.FileStat{}, fmt.Errorf("%s: %w", path, err)
	}

	// A note that is not there cannot hold anything the caller has not read, so it is made.
	if looked == nil && e.Baseline.isStale(on, markdown.Normalise(doc.Body())) {
		return domain.FileStat{}, fmt.Errorf("write %s: %w", path, ErrStale)
	}

	if err := change(doc); err != nil {
		return domain.FileStat{}, fmt.Errorf("%s: %w", path, err)
	}
	if err := e.addIdentifier(doc); err != nil {
		return domain.FileStat{}, err
	}

	// Every change to a note's contents comes through here, so the size it is written within is asked
	// once, of the file the change came to.
	content := doc.Bytes()
	if err := CheckSize(path, len(content), e.Bound); err != nil {
		return domain.FileStat{}, err
	}
	return files.Write(ctx, path, content, against)
}

// statNote is the file stat of the file on disk, and the error of asking, for a caller that holds
// the note to what it last saw or to what is there now.
func (e Edit) statNote(ctx context.Context, files Reader, path string) (domain.FileStat, error) {
	if e.Baseline != nil || (e.FileStat.IsZero() && !e.ShouldOverwrite) {
		return files.StatFile(ctx, path)
	}
	return domain.FileStat{}, nil
}

// readNote is the bytes of the note at path. A caller overwriting a note that is not there gets no
// bytes: the write makes the note out of the person's own text, with no frontmatter and no
// identifier.
func (e Edit) readNote(ctx context.Context, files Reader, path string) ([]byte, error) {
	raw, err := files.Read(ctx, path)
	switch {
	case err == nil:
		return raw, nil
	case e.ShouldOverwrite && errors.Is(err, fs.ErrNotExist):
		return nil, nil
	default:
		return nil, fmt.Errorf("read %s: %w", path, MapMissingNote(err))
	}
}

func (e Edit) getExpected(path string, on domain.FileStat, looked error) (domain.FileStat, error) {
	if !e.FileStat.IsZero() || e.ShouldOverwrite {
		return e.FileStat, nil
	}
	if looked != nil {
		return domain.FileStat{}, fmt.Errorf("look at %s: %w", path, MapMissingNote(looked))
	}
	if on.Kind != "" && on.Kind != domain.KindNote {
		return domain.FileStat{}, fmt.Errorf("look at %s: %w", path, ErrNotANote)
	}
	return on, nil
}

// addIdentifier writes the identifier a note arrived without. A caller writing over what is there
// leaves the frontmatter as the person wrote it.
func (e Edit) addIdentifier(doc *markdown.Document) error {
	if _, carried := doc.Identifier(); carried || e.ShouldOverwrite {
		return nil
	}
	return doc.SetIdentifier(ulid.New(e.Now(), e.Entropy))
}
