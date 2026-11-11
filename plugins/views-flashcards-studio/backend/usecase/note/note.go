// Package note is what the use cases of the plugin do to a note held in a vault: reading its prose,
// editing it, pointing it at another note and listing what it is joined to.
//
// The rules are the notes module's; this package puts them over the ports of the plugin.
package note

import (
	"context"
	"crypto/rand"
	"errors"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// The notes module's words for a read and an edit.
type (
	// ReadOutcome is how a read ended.
	ReadOutcome = edit.ReadOutcome
	// Contents is one note as returned by a read.
	Contents = edit.Contents
	// IndexUpdater brings the named notes up to date in the index.
	IndexUpdater = edit.IndexUpdater
	// Edit is what every change to the contents of an existing note needs.
	Edit = edit.Edit
)

const (
	// MaxBytes is the most a note may be and still be read here.
	MaxBytes = edit.MaxBytes
)

// The outcomes of reading a note.
const (
	Ok         = edit.Ok
	Missing    = edit.Missing
	NotANote   = edit.NotANote
	NotText    = edit.NotText
	TooLarge   = edit.TooLarge
	Unreadable = edit.Unreadable
)

// The errors of reading and writing a note.
var (
	ErrNoNote         = edit.ErrNoNote
	ErrIndexOutdated  = edit.ErrIndexOutdated
	ErrTooLarge       = edit.ErrTooLarge
	ErrBodyUnwritable = edit.ErrBodyUnwritable
)

// WrapIndexOutdated wraps err as a failure to bring the index up to date after a
// write. A caller distinguishes it from a write that never landed.
func WrapIndexOutdated(err error, paths ...string) error {
	return edit.WrapIndexOutdated(err, paths...)
}

// CheckSize holds a file to the most it may be. Zero holds it to nothing.
func CheckSize(path string, size, limit int) error { return edit.CheckSize(path, size, limit) }

// GetAddress is how a note known by path is written into a link.
func GetAddress(ctx context.Context, names ports.NameQueries, vaultID domain.VaultID, path string) (domain.Address, error) {
	return edit.GetAddress(ctx, names, vaultID, path)
}

// NewEdit is what a change to an existing note is made through: the vault it is read and written
// through, what updates the index for it, and what time it is. The function returned with it
// returns the failures to release the write lock of the changes made through it, which edit.Files
// releases without an error.
func NewEdit(readers ports.VaultReaders, writers ports.VaultWriters, index IndexUpdater, now ports.Clock) (Edit, func() error) {
	failures := &releaseFailures{mu: sync.Mutex{}, err: nil}
	return edit.Edit{
		Open: func(v domain.Vault) (edit.Files, error) {
			reader, err := readers.Open(v)
			if err != nil {
				return nil, err
			}
			writer, err := writers.Open(v)
			if err != nil {
				return nil, err
			}
			return files{reader: reader, writer: writer, writers: writers, vault: v, failures: failures}, nil
		},
		Index:           index,
		Clock:           now,
		Entropy:         rand.Reader,
		FileStat:        domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}},
		ShouldOverwrite: false,
		Bound:           0,
		Baseline:        nil,
	}, failures.getError
}

// files are the files of one vault, read and written through the ports.
type files struct {
	reader   ports.VaultReader
	writer   ports.VaultWriter
	writers  ports.VaultWriters
	vault    domain.Vault
	failures *releaseFailures
}

// releaseFailures joins the failures to release a write lock.
type releaseFailures struct {
	mu  sync.Mutex
	err error
}

func (r *releaseFailures) addError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.err = errors.Join(r.err, err)
}

func (r *releaseFailures) getError() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

func (f files) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	return f.reader.StatFile(ctx, path)
}

func (f files) Read(ctx context.Context, path string) ([]byte, error) {
	return f.reader.Read(ctx, path)
}

func (f files) Write(ctx context.Context, path string, content []byte, stat domain.FileStat) (domain.FileStat, error) {
	return f.writer.Write(ctx, path, content, stat)
}

func (f files) Lock(ctx context.Context) (func(), error) {
	release, err := f.writers.Lock(ctx, f.vault)
	if err != nil {
		return nil, err
	}
	return func() { f.failures.addError(release()) }, nil
}

// Read returns the prose of one note.
type Read struct {
	readers ports.VaultReaders
}

// NewRead is the reader of the prose of a note.
func NewRead(readers ports.VaultReaders) Read { return Read{readers: readers} }

// Execute reads the note at path. An error is the vault being out of reach; what is wrong with the
// note itself is an outcome.
func (u Read) Execute(ctx context.Context, v domain.Vault, path string) (Contents, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return Contents{}, err
	}
	return edit.Read(ctx, reader, path)
}

// Write replaces the prose of a note and leaves its frontmatter alone.
type Write struct {
	readers ports.VaultReaders
	writers ports.VaultWriters
	index   IndexUpdater
	now     ports.Clock
	Bound   int
}

// NewWrite is the writer a note goes to disk through.
func NewWrite(readers ports.VaultReaders, writers ports.VaultWriters, index IndexUpdater, now ports.Clock) Write {
	return Write{readers: readers, writers: writers, index: index, now: now, Bound: 0}
}

// Execute puts body in the note at path over the file stat the caller read, and returns the
// file stat of the file this write produced.
func (u Write) Execute(
	ctx context.Context, v domain.Vault, path, body string, stat domain.FileStat,
) (domain.FileStat, error) {
	if markdown.IsFrontmatterStart(body) {
		return domain.FileStat{}, ErrBodyUnwritable
	}
	e, getReleaseError := NewEdit(u.readers, u.writers, u.index, u.now)
	e.FileStat = stat
	e.Bound = u.Bound
	if e.Bound == 0 {
		e.Bound = MaxBytes
	}
	written, err := e.Apply(ctx, v, path, func(doc *markdown.Document) error { return doc.SetBody(body) })
	return written, errors.Join(err, getReleaseError())
}

// EditLinks writes the relationships one note declares.
type EditLinks struct {
	readers ports.VaultReaders
	writers ports.VaultWriters
	index   IndexUpdater
	now     ports.Clock
}

// NewEditLinks is what a note's relationships are written through.
func NewEditLinks(readers ports.VaultReaders, writers ports.VaultWriters, index IndexUpdater, now ports.Clock) EditLinks {
	return EditLinks{readers: readers, writers: writers, index: index, now: now}
}

// LinkEntry is the entry of the `links:` block of a note that PointAt writes: the link `type` it is
// found under, the place it names and the role of the link. An empty To takes the entry out.
type LinkEntry struct {
	Type string
	To   domain.Address
	Role domain.LinkRole
}

// PointAt makes the note at from name one place under a link `type`, over the file stat the caller
// read.
func (u EditLinks) PointAt(
	ctx context.Context, v domain.Vault, from string, entry LinkEntry, stat domain.FileStat,
) (domain.FileStat, error) {
	e, getReleaseError := NewEdit(u.readers, u.writers, u.index, u.now)
	e.FileStat = stat
	written, err := e.PointAt(ctx, v, from, entry.Type, entry.To, entry.Role)
	return written, errors.Join(err, getReleaseError())
}

// Links is what one note is joined to: what it points at, and what points at it.
type Links struct {
	Links     []domain.ResolvedLink
	Backlinks []domain.ResolvedLink
}

// ShowLinks gathers them for one note.
type ShowLinks struct {
	links ports.LinkQueries
}

// NewShowLinks returns a ShowLinks that serves both halves from links.
func NewShowLinks(links ports.LinkQueries) ShowLinks { return ShowLinks{links: links} }

// Execute returns both halves of what the note is joined to.
func (u ShowLinks) Execute(ctx context.Context, v domain.Vault, path string) (Links, error) {
	links, err := u.links.GetLinks(ctx, v.ID, path)
	if err != nil {
		return Links{}, err
	}
	backlinks, err := u.links.GetBacklinks(ctx, v.ID, path)
	if err != nil {
		return Links{}, err
	}
	return Links{Links: links, Backlinks: backlinks}, nil
}
