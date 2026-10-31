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
	// Write replaces the file if it still matches the file stat, else it returns ErrStale.
	// An empty file stat disables the staleness check.
	Write(ctx context.Context, path string, content []byte, stat domain.FileStat) (domain.FileStat, error)
	// Lock holds the vault's writers until the release is called.
	Lock(ctx context.Context) (release func(), err error)
}

// Levels updates the named notes in the index. A write is reported complete
// after its changes are searchable.
type Levels func(ctx context.Context, v domain.Vault, paths []string) error

// Baseline is what a caller last saw of the note it is saving.
//
// Prose is compared first: text equal to the text the caller read is unchanged,
// regardless of file size and time. A synchroniser, a checkout and a touch all change those
// without changing the text, and comparing them alone would report a conflict for an unedited file.
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

// Clock gets the current time.
type Clock interface {
	Now() time.Time
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
	// Clock gives the time an identifier is minted at.
	Clock Clock
	// Entropy is the randomness source for minted identifiers.
	Entropy io.Reader

	FileStat domain.FileStat
	// ShouldOverwrite is set by a caller writing the user's current text: the note on disk is
	// replaced and no identifier is added.
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

	// The file stat is read before the file bytes, matching the order in which a read
	// returns them.
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

	// Every change to a note's contents passes through here, so the size limit is checked
	// once, against the content being written.
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

// readNote returns the bytes of the note at path. A caller overwriting a note that does not
// exist gets no bytes: the write creates the note from the user's own text, with no
// frontmatter and no identifier.
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

// addIdentifier writes an identifier into a note that has none. A caller overwriting
// existing content leaves the frontmatter unchanged.
func (e Edit) addIdentifier(doc *markdown.Document) error {
	if _, carried := doc.Identifier(); carried || e.ShouldOverwrite {
		return nil
	}
	return doc.SetIdentifier(ulid.New(e.Clock.Now(), e.Entropy))
}
