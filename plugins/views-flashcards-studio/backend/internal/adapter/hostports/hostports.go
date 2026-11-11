// Package hostports answers the ports of the use cases from the vault the host lets the plugin work.
package hostports

import (
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// reviewLogSuffix is what a file of review answers is named with.
const reviewLogSuffix = ".jsonl"

// Ports is every port the use cases take, answered by one host vault and the plugin's folder of its
// store.
type Ports struct {
	Readers   ports.VaultReaders
	Writers   ports.VaultWriters
	Notes     Notes
	Links     ports.LinkQueries
	Problems  ports.ProblemQueries
	Logs      ports.LogStores
	Schedules func(name string) ports.ScheduleStore
}

// Vault is the calls the ports make on one vault, each through the narrow interface it needs.
type Vault struct {
	Files    host.FileReader
	Writer   host.FileWriter
	Index    host.VaultIndex
	Notes    host.NoteLookup
	Links    host.LinkLookup
	Cache    host.CacheStore
	Selector host.VaultSelector
}

// New returns the ports over the host vault and the plugin's folder. A call on a
// vault names it by identity, so the same ports serve a window for one vault and a
// window that counts all of them.
func New(vaults func(domain.VaultID) Vault, store func(domain.VaultID) Folder) Ports {
	return Ports{
		Readers:   Readers{Vaults: vaults},
		Writers:   Writers{Vaults: vaults},
		Notes:     Notes{Vaults: vaults},
		Links:     Links{Vaults: vaults},
		Problems:  Problems{Vaults: vaults},
		Logs:      Logs{Store: store},
		Schedules: func(name string) ports.ScheduleStore { return Schedules{Vaults: vaults, Name: name} },
	}
}

// Readers opens the files of a vault for reading.
type Readers struct{ Vaults func(domain.VaultID) Vault }

// Open returns the reader of the vault v.
func (r Readers) Open(v domain.Vault) (ports.VaultReader, error) {
	return reader{vault: r.Vaults(v.ID).Files}, nil
}

type reader struct{ vault host.FileReader }

func (r reader) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	pos, err := r.vault.StatFile(ctx, path)
	return pos, readError(err)
}

func (r reader) Read(ctx context.Context, path string) ([]byte, error) {
	content, _, err := r.vault.Read(ctx, path)
	return content, readError(err)
}

// Writers opens the files of a vault for writing.
type Writers struct{ Vaults func(domain.VaultID) Vault }

// Open returns the writer of the vault v.
func (w Writers) Open(v domain.Vault) (ports.VaultWriter, error) {
	return writer{vault: w.Vaults(v.ID).Writer}, nil
}

// Lock takes the write lock of the vault v and returns the function that releases it.
func (w Writers) Lock(ctx context.Context, v domain.Vault) (func() error, error) {
	return w.Vaults(v.ID).Writer.Lock(ctx)
}

type writer struct{ vault host.FileWriter }

func (w writer) Write(ctx context.Context, path string, content []byte, stat domain.FileStat) (domain.FileStat, error) {
	pos, err := w.vault.Write(ctx, path, content, stat)
	return pos, readError(err)
}

func (w writer) Create(ctx context.Context, path string, content []byte) error {
	return readError(w.vault.Create(ctx, path, content))
}

func readError(err error) error {
	switch {
	case errors.Is(err, host.ErrAlreadyExists):
		return ports.ErrAlreadyExists
	case errors.Is(err, host.ErrLockedByAnother):
		return ports.ErrLockedByAnother
	}
	return err
}

// Notes answers queries about a vault's notes from the index.
type Notes struct{ Vaults func(domain.VaultID) Vault }

// GetNotes returns the notes at paths.
func (n Notes) GetNotes(ctx context.Context, id domain.VaultID, paths []string) (map[string]domain.NoteRef, error) {
	return n.Vaults(id).Notes.GetNotes(ctx, paths)
}

// GetTypes returns the note types of the notes at paths.
func (n Notes) GetTypes(ctx context.Context, id domain.VaultID, paths []string) (map[string]domain.NoteType, error) {
	return n.Vaults(id).Notes.GetTypes(ctx, paths)
}

// GetPathsOfType returns the paths of the notes of a type.
func (n Notes) GetPathsOfType(ctx context.Context, id domain.VaultID, noteType domain.NoteType) ([]string, error) {
	return n.Vaults(id).Notes.GetPathsOfType(ctx, noteType)
}

// GetCardModels lists the notes of the card model type with their titles, by path.
func (n Notes) GetCardModels(ctx context.Context, id domain.VaultID) ([]ports.CardModel, error) {
	vault := n.Vaults(id).Notes
	paths, err := vault.GetPathsOfType(ctx, format.TypeCardModel)
	if err != nil {
		return nil, err
	}
	titled, err := vault.GetNotes(ctx, paths)
	if err != nil {
		return nil, err
	}
	out := make([]ports.CardModel, 0, len(paths))
	for _, path := range paths {
		out = append(out, ports.CardModel{Path: path, Title: titled[path].Title})
	}
	return out, nil
}

// GetNamedPaths returns the paths of the notes that a name resolves to.
func (n Notes) GetNamedPaths(ctx context.Context, id domain.VaultID, name string) ([]string, error) {
	return n.Vaults(id).Notes.GetNamedPaths(ctx, name)
}

// Has reports whether the index contains the vault.
func (n Notes) Has(ctx context.Context, id domain.VaultID) (bool, error) {
	state, err := n.Vaults(id).Index.EnsureIndexed(ctx)
	if errors.Is(err, host.ErrUnknownVault) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return state.Status == host.Indexed, nil
}

// Links answers queries about a vault's links from the index.
type Links struct{ Vaults func(domain.VaultID) Vault }

// GetLinks returns the links that the note at from contains.
func (l Links) GetLinks(ctx context.Context, id domain.VaultID, from string) ([]domain.ResolvedLink, error) {
	return l.Vaults(id).Links.GetLinks(ctx, from)
}

// GetBacklinks returns the links that point at the note at to.
func (l Links) GetBacklinks(ctx context.Context, id domain.VaultID, to string) ([]domain.ResolvedLink, error) {
	return l.Vaults(id).Links.GetBacklinks(ctx, to)
}

// ResolveLinks resolves rawLinks as written in the note at from.
func (l Links) ResolveLinks(ctx context.Context, id domain.VaultID, from string, rawLinks []string) (map[string]domain.ResolvedLink, error) {
	return l.Vaults(id).Links.ResolveLinks(ctx, from, rawLinks)
}

// Problems answers what parsing a vault's files turned up.
type Problems struct{ Vaults func(domain.VaultID) Vault }

// GetParseProblems returns the parse problems of the vault.
func (p Problems) GetParseProblems(ctx context.Context, id domain.VaultID) ([]domain.VaultProblem, error) {
	return p.Vaults(id).Index.GetParseProblems(ctx)
}

// Folder is the plugin's folder in the store of a vault, as the review logs use it.
type Folder interface {
	Read(ctx context.Context, name string) ([]byte, error)
	Append(ctx context.Context, name string, content []byte) error
	List(ctx context.Context) ([]host.StoreFile, error)
}

// Logs opens the plugin's folder of a vault's store, which the answers are kept in.
type Logs struct{ Store func(domain.VaultID) Folder }

// Open returns the log store of the vault v.
func (l Logs) Open(v domain.Vault) (ports.LogStore, error) {
	return store{folder: l.Store(v.ID)}, nil
}

type store struct{ folder Folder }

func fileOf(run string) string { return run + reviewLogSuffix }

func (s store) ReadLog(ctx context.Context, run string) ([]byte, error) {
	content, err := s.folder.Read(ctx, fileOf(run))
	return content, readError(err)
}

func (s store) AppendLog(ctx context.Context, run string, content []byte) error {
	return readError(s.folder.Append(ctx, fileOf(run), content))
}

func (s store) ListLogs(ctx context.Context) ([]ports.Entry, error) {
	target, err := s.folder.List(ctx)
	if err != nil {
		return nil, readError(err)
	}
	out := make([]ports.Entry, 0, len(target))
	for _, one := range target {
		id, isLog := strings.CutSuffix(one.Name, reviewLogSuffix)
		if !isLog {
			continue
		}
		out = append(out, ports.Entry{Run: id, Size: int(one.Size)})
	}
	return out, nil
}

// Schedules keeps what was worked out for a vault in the cache the host holds for the plugin, under
// a name.
type Schedules struct {
	Vaults func(domain.VaultID) Vault
	Name   string
}

func (s Schedules) Read(ctx context.Context, id domain.VaultID) ([]byte, error) {
	content, found, err := s.Vaults(id).Cache.ReadCache(ctx, s.Name)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fs.ErrNotExist
	}
	return content, nil
}

func (s Schedules) Write(ctx context.Context, id domain.VaultID, content []byte) error {
	return s.Vaults(id).Cache.WriteCache(ctx, s.Name, content)
}
