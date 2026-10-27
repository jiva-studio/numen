// Package hostports answers the ports of the use cases from the vault the host lets the plugin work.
package hostports

import (
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// reviewLogSuffix is what a file of review answers is named with.
const reviewLogSuffix = ".jsonl"

// Ports is every port the use cases take, answered by one host vault and the plugin's folder of its
// store.
type Ports struct {
	Readers   ports.VaultReaders
	Writers   ports.VaultWriters
	Notes     ports.NoteQueries
	Links     ports.LinkQueries
	Problems  ports.ProblemQueries
	Logs      ports.LogStores
	Schedules func(name string) ports.ScheduleStore
}

// New answers the ports from the host vault and the plugin's folder. A call on a vault names it by
// identity, so that the same ports serve a window showing one vault and a window counting them all.
func New(vault host.Vault, store host.Store) Ports {
	return Ports{
		Readers:   Readers{Vault: vault},
		Writers:   Writers{Vault: vault},
		Notes:     Notes{Vault: vault},
		Links:     Links{Vault: vault},
		Problems:  Problems{Vault: vault},
		Logs:      Logs{Store: store},
		Schedules: func(name string) ports.ScheduleStore { return Schedules{Vault: vault, Name: name} },
	}
}

// Readers opens the files of a vault for reading.
type Readers struct{ Vault host.Vault }

func (r Readers) Open(v domain.Vault) (ports.VaultReader, error) {
	return reader{vault: r.Vault.With(v.ID)}, nil
}

type reader struct{ vault host.Vault }

func (r reader) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	pos, err := r.vault.StatFile(ctx, path)
	return pos, readError(err)
}

func (r reader) Read(ctx context.Context, path string) ([]byte, error) {
	content, _, err := r.vault.Read(ctx, path)
	return content, readError(err)
}

// Writers opens the files of a vault for writing.
type Writers struct{ Vault host.Vault }

func (w Writers) Open(v domain.Vault) (ports.VaultWriter, error) {
	return writer{vault: w.Vault.With(v.ID)}, nil
}

func (w Writers) Lock(ctx context.Context, v domain.Vault) (func(), error) {
	return w.Vault.With(v.ID).Lock(ctx)
}

type writer struct{ vault host.Vault }

func (w writer) Write(ctx context.Context, path string, content []byte, stat domain.FileStat) (domain.FileStat, error) {
	pos, err := w.vault.Write(ctx, path, content, stat)
	return pos, readError(err)
}

func (w writer) Create(ctx context.Context, path string, content []byte) error {
	return readError(w.vault.Create(ctx, path, content))
}

func readError(err error) error {
	switch {
	case errors.Is(err, host.ErrOccupied):
		return ports.ErrOccupied
	case errors.Is(err, host.ErrHeldByAnother):
		return ports.ErrHeldByAnother
	}
	return err
}

// Notes answers what the index says of a vault's notes.
type Notes struct{ Vault host.Vault }

func (n Notes) GetNotes(ctx context.Context, id domain.VaultID, paths []string) (map[string]domain.NoteRef, error) {
	return n.Vault.With(id).GetNotes(ctx, paths)
}

func (n Notes) GetTypes(ctx context.Context, id domain.VaultID, paths []string) (map[string]domain.NoteType, error) {
	return n.Vault.With(id).GetTypes(ctx, paths)
}

func (n Notes) GetPathsOfType(ctx context.Context, id domain.VaultID, noteType domain.NoteType) ([]string, error) {
	return n.Vault.With(id).GetPathsOfType(ctx, noteType)
}

// GetCardModels lists the notes of the card model type with their titles, by path.
func (n Notes) GetCardModels(ctx context.Context, id domain.VaultID) ([]ports.CardModel, error) {
	vault := n.Vault.With(id)
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

func (n Notes) GetNamedPaths(ctx context.Context, id domain.VaultID, name string) ([]string, error) {
	return n.Vault.With(id).GetNamedPaths(ctx, name)
}

// Has says whether the index carries the vault.
func (n Notes) Has(ctx context.Context, id domain.VaultID) (bool, error) {
	state, err := n.Vault.With(id).EnsureIndexed(ctx)
	if errors.Is(err, host.ErrUnknownVault) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return state.Status == host.Indexed, nil
}

// Links answers what the index says of a vault's links.
type Links struct{ Vault host.Vault }

func (l Links) GetLinks(ctx context.Context, id domain.VaultID, from string) ([]domain.ResolvedLink, error) {
	return l.Vault.With(id).GetLinks(ctx, from)
}

func (l Links) GetBacklinks(ctx context.Context, id domain.VaultID, to string) ([]domain.ResolvedLink, error) {
	return l.Vault.With(id).GetBacklinks(ctx, to)
}

func (l Links) ResolveLinks(ctx context.Context, id domain.VaultID, from string, rawLinks []string) (map[string]domain.ResolvedLink, error) {
	return l.Vault.With(id).ResolveLinks(ctx, from, rawLinks)
}

// Problems answers what parsing a vault's files turned up.
type Problems struct{ Vault host.Vault }

func (p Problems) GetParseProblems(ctx context.Context, id domain.VaultID) ([]domain.VaultProblem, error) {
	return p.Vault.With(id).GetParseProblems(ctx)
}

// Logs opens the plugin's folder of a vault's store, which the answers are kept in.
type Logs struct{ Store host.Store }

func (l Logs) Open(v domain.Vault) (ports.LogStore, error) {
	return store{folder: l.Store.With(v.ID)}, nil
}

type store struct{ folder host.Store }

// fileOf is the file the answers of a run are kept in.
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
	Vault host.Vault
	Name  string
}

func (s Schedules) Read(ctx context.Context, id domain.VaultID) ([]byte, error) {
	content, found, err := s.Vault.With(id).ReadCache(ctx, s.Name)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fs.ErrNotExist
	}
	return content, nil
}

func (s Schedules) Write(ctx context.Context, id domain.VaultID, content []byte) error {
	return s.Vault.With(id).WriteCache(ctx, s.Name, content)
}
