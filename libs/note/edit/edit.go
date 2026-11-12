package edit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/markdown"
	"github.com/jiva-studio/numen/libs/note/ulid"
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

// IndexUpdater updates the named notes in the index. A write is reported complete
// after its changes are searchable.
type IndexUpdater func(ctx context.Context, v domain.Vault, paths []string) error

// Baseline is what a caller last saw of the note it is saving.
// Prose is compared first: equal text is unchanged whatever the file size and
// time, which a synchronizer, a checkout or a touch change alone.
// FileStat answers for text that did move: a note at the file stat of the last
// write is the note this caller put there.
type Baseline struct {
	Prose    string
	FileStat domain.FileStat
}

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
// the one thing, put it back, and update the index.
type Edit struct {
	Open    func(v domain.Vault) (Files, error)
	Index   IndexUpdater
	Clock   Clock
	Entropy io.Reader

	FileStat        domain.FileStat
	ShouldOverwrite bool
	Bound           int
	Baseline        *Baseline
}

// Apply makes one change to the note at path and puts it back.
func (e Edit) Apply(ctx context.Context, v domain.Vault, path string, change func(*markdown.Document) error) (domain.FileStat, error) {
	content, err := e.splice(ctx, v, path, change)
	if err != nil {
		return domain.FileStat{}, err
	}
	// The file is on disk, so the file stat is stored alongside whatever the index update produced and a
	// caller can tell the two apart.
	if e.Index == nil {
		return content, nil
	}
	// The file is written, so the index follows it even when the caller is canceled.
	return content, WrapIndexOutdated(e.Index(context.WithoutCancel(ctx), v, []string{path}), path)
}

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
	against, err := e.getExpectedStat(path, on, looked)
	if err != nil {
		return domain.FileStat{}, err
	}
	doc, err := markdown.Open(raw)
	if err != nil {
		return domain.FileStat{}, fmt.Errorf("%s: %w", path, err)
	}

	// A note that is not there cannot hold anything the caller has not read, so it is made.
	if looked == nil && e.Baseline.isStale(on, markdown.Normalize(doc.Body())) {
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

func (e Edit) statNote(ctx context.Context, files Reader, path string) (domain.FileStat, error) {
	if e.Baseline != nil || (e.FileStat.IsZero() && !e.ShouldOverwrite) {
		return files.StatFile(ctx, path)
	}
	return domain.FileStat{}, nil
}

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

func (e Edit) getExpectedStat(path string, on domain.FileStat, looked error) (domain.FileStat, error) {
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

func (e Edit) addIdentifier(doc *markdown.Document) error {
	if _, carried := doc.Identifier(); carried || e.ShouldOverwrite {
		return nil
	}
	identifier, err := ulid.NewGenerator(e.Entropy).NewID(e.Clock.Now())
	if err != nil {
		return err
	}
	return doc.SetIdentifier(identifier)
}
