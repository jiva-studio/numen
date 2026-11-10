package hostvault

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"

	noteport "github.com/jiva-studio/numen/libs/core/kernel/usecase/note/notecontract"
	problemport "github.com/jiva-studio/numen/libs/core/kernel/usecase/note/problemcontract"
	pluginport "github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand/plugincontract"
	vaultport "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/vaultcontract"
)

var ErrNoVault = errors.New("the host is showing no vault")

// ErrNoConnections is returned when the host was built without the operations on
// connections.
var ErrNoConnections = errors.New("the host has no operations on connections")

// NoteWriter writes and creates the notes of one vault.
type NoteWriter interface {
	Write(ctx context.Context, path string, content []byte, stat domain.FileStat) (domain.FileStat, error)
	Create(ctx context.Context, path string, content []byte) error
}

// Vault is port.PluginVault over the ports of the installation.
type Vault struct {
	Current     func() (domain.Vault, bool)
	Readers     vaultport.VaultReaders
	OpenWriter  func(v domain.Vault) (NoteWriter, error)
	LockVault   func(ctx context.Context, v domain.Vault) (func(), error)
	Notes       noteRefs
	Types       noteTypes
	NamedPaths  namedPaths
	TypedPaths  typedPaths
	Links       noteport.LinkQueries
	Connections Connections
	Problems    problemport.ProblemQueries
	Arranger    Arranger
	Stores      pluginport.PluginStores
	Cache       pluginport.PluginCache
	Watcher     vaultport.VaultWatcher
	Vaults      func() ([]domain.Vault, error)
	Find        func(id domain.VaultID) (domain.Vault, error)
	IndexState  func(ctx context.Context, v domain.Vault) (plugin.IndexState, error)
	Moved       func(ctx context.Context) (<-chan struct{}, error)
	Written     func()
	Index       func(ctx context.Context, v domain.Vault, paths []string) error
}

func (a Vault) GetVault() (domain.Vault, bool) {
	if a.Current == nil {
		return domain.Vault{}, false
	}
	return a.Current()
}

func (a Vault) GetVaults() ([]domain.Vault, error) {
	if a.Vaults != nil {
		return a.Vaults()
	}
	if v, showing := a.GetVault(); showing {
		return []domain.Vault{v}, nil
	}
	return nil, nil
}

func (a Vault) With(id domain.VaultID) (pluginport.PluginVault, error) {
	found, err := a.find(id)
	if err != nil {
		return nil, err
	}
	linked := a
	linked.Current = func() (domain.Vault, bool) { return found, true }
	return linked, nil
}

func (a Vault) find(id domain.VaultID) (domain.Vault, error) {
	if a.Find != nil {
		return a.Find(id)
	}
	if v, showing := a.GetVault(); showing && v.ID == id {
		return v, nil
	}
	return domain.Vault{}, fmt.Errorf("vault %q: %w", id, fs.ErrNotExist)
}

func (a Vault) EnsureIndexed(ctx context.Context) (plugin.IndexState, error) {
	v, err := a.open()
	if err != nil {
		return plugin.IndexState{}, err
	}
	if a.IndexState == nil {
		return plugin.IndexState{Status: plugin.Indexed}, nil
	}
	return a.IndexState(ctx, v)
}

func (a Vault) WatchMoves(ctx context.Context) (<-chan struct{}, error) {
	if a.Moved == nil {
		return nil, fmt.Errorf("watch moves: %w", domain.ErrUnavailable)
	}
	return a.Moved(ctx)
}

func (a Vault) open() (domain.Vault, error) {
	v, showing := a.GetVault()
	if !showing {
		return domain.Vault{}, ErrNoVault
	}
	return v, nil
}

func (a Vault) openReader() (vaultport.VaultReader, error) {
	v, err := a.open()
	if err != nil {
		return nil, err
	}
	return a.Readers.Open(v)
}

func (a Vault) openWriter() (NoteWriter, error) {
	v, err := a.open()
	if err != nil {
		return nil, err
	}
	return a.OpenWriter(v)
}

func (a Vault) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	reader, err := a.openReader()
	if err != nil {
		return domain.FileStat{}, err
	}
	return reader.StatFile(ctx, path)
}

func (a Vault) Read(ctx context.Context, path string) ([]byte, domain.FileStat, error) {
	reader, err := a.openReader()
	if err != nil {
		return nil, domain.FileStat{}, err
	}
	pos, err := reader.StatFile(ctx, path)
	if err != nil {
		return nil, domain.FileStat{}, err
	}
	content, err := reader.Read(ctx, path)
	if err != nil {
		return nil, domain.FileStat{}, err
	}
	return content, pos, nil
}

// ReadRange reads the range through the read of usecase/file, with the file stat taken before the
// bytes are read.
func (a Vault) ReadRange(ctx context.Context, path string, start, length int) (plugin.FileRange, error) {
	v, err := a.open()
	if err != nil {
		return plugin.FileRange{}, err
	}
	reader, err := a.Readers.Open(v)
	if err != nil {
		return plugin.FileRange{}, err
	}
	pos, err := reader.StatFile(ctx, path)
	if errors.Is(err, vaultport.ErrNotANote) {
		return plugin.FileRange{}, fmt.Errorf("%s is not a file in the vault: %w", path, fs.ErrNotExist)
	}
	if err != nil {
		return plugin.FileRange{}, err
	}
	if length <= 0 || length > file.MostRead {
		length = file.MostRead
	}
	result, err := file.Read{Readers: a.Readers}.Execute(ctx, v, path, start, length)
	if err != nil {
		return plugin.FileRange{}, err
	}
	switch result.Outcome {
	case file.Ok:
		return plugin.FileRange{Text: result.Text, Start: result.Start, Stat: pos}, nil
	case file.NotText:
		return plugin.FileRange{}, fmt.Errorf("read %s from %d: %w", path, result.Start, plugin.ErrNotText)
	case file.Missing, file.AFolder, file.LeftAlone:
	}
	return plugin.FileRange{}, fmt.Errorf("%s is %s: %w", path, result.Outcome, fs.ErrNotExist)
}

func (a Vault) Write(
	ctx context.Context, path string, content []byte, baseline domain.FileStat,
) (domain.FileStat, error) {
	writer, err := a.openWriter()
	if err != nil {
		return domain.FileStat{}, err
	}
	pos, err := writer.Write(ctx, path, content, baseline)
	if err != nil {
		return pos, err
	}
	if err := a.Scan(ctx, []string{path}); err != nil {
		return pos, fmt.Errorf("index %s after write: %w", path, err)
	}
	return pos, nil
}

func (a Vault) notifyFileWritten() {
	if a.Written != nil {
		a.Written()
	}
}

func (a Vault) Create(ctx context.Context, path string, content []byte) error {
	writer, err := a.openWriter()
	if err != nil {
		return err
	}
	if err := writer.Create(ctx, path, content); err != nil {
		return err
	}
	if err := a.Scan(ctx, []string{path}); err != nil {
		return fmt.Errorf("index %s after create: %w", path, err)
	}
	return nil
}

func (a Vault) Lock(ctx context.Context) (func(), error) {
	v, err := a.open()
	if err != nil {
		return nil, err
	}
	return a.LockVault(ctx, v)
}

func (a Vault) Scan(ctx context.Context, paths []string) error {
	v, err := a.open()
	if err != nil {
		return err
	}
	a.notifyFileWritten()
	if a.Index == nil {
		return nil
	}
	return a.Index(ctx, v, paths)
}

func (a Vault) GetNotes(ctx context.Context, paths []string) (map[string]domain.NoteRef, error) {
	v, err := a.open()
	if err != nil {
		return nil, err
	}
	return a.Notes.GetNotes(ctx, v.ID, paths)
}

func (a Vault) GetTypes(ctx context.Context, paths []string) (map[string]domain.NoteType, error) {
	v, err := a.open()
	if err != nil {
		return nil, err
	}
	return a.Types.GetTypes(ctx, v.ID, paths)
}

func (a Vault) GetNamedPaths(ctx context.Context, name string) ([]string, error) {
	v, err := a.open()
	if err != nil {
		return nil, err
	}
	return a.NamedPaths.GetNamedPaths(ctx, v.ID, name)
}

func (a Vault) GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error) {
	v, err := a.open()
	if err != nil {
		return nil, err
	}
	return a.TypedPaths.GetPathsOfType(ctx, v.ID, noteType)
}

func (a Vault) GetLinks(ctx context.Context, from string) ([]domain.ResolvedLink, error) {
	v, err := a.open()
	if err != nil {
		return nil, err
	}
	return a.Links.GetLinks(ctx, v.ID, from)
}

func (a Vault) GetBacklinks(ctx context.Context, to string) ([]domain.ResolvedLink, error) {
	v, err := a.open()
	if err != nil {
		return nil, err
	}
	return a.Links.GetBacklinks(ctx, v.ID, to)
}

func (a Vault) ResolveLinks(
	ctx context.Context, from string, targets []string,
) (map[string]domain.ResolvedLink, error) {
	v, err := a.open()
	if err != nil {
		return nil, err
	}
	return a.Links.ResolveLinks(ctx, v.ID, from, targets)
}

func (a Vault) GetParseProblems(ctx context.Context) ([]domain.VaultProblem, error) {
	v, err := a.open()
	if err != nil || a.Problems == nil {
		return nil, err
	}
	return a.Problems.GetParseProblems(ctx, v.ID)
}

func (a Vault) OpenStore(pluginID string) (pluginport.PluginStore, error) {
	v, err := a.open()
	if err != nil {
		return nil, err
	}
	return a.Stores.Open(v, pluginID)
}

func (a Vault) ReadCache(ctx context.Context, pluginID, name string) ([]byte, bool, error) {
	v, err := a.open()
	if err != nil {
		return nil, false, err
	}
	return a.Cache.Read(ctx, pluginID, v.ID, name)
}

func (a Vault) WriteCache(ctx context.Context, pluginID, name string, content []byte) error {
	v, err := a.open()
	if err != nil {
		return err
	}
	return a.Cache.Write(ctx, pluginID, v.ID, name, content)
}

func (a Vault) Watch(ctx context.Context) (<-chan []string, <-chan struct{}, error) {
	v, err := a.open()
	if err != nil {
		return nil, nil, err
	}
	if a.Watcher == nil {
		return nil, nil, fmt.Errorf("watch %s: %w", v.ID, domain.ErrUnavailable)
	}
	return a.Watcher.WatchVault(ctx, v)
}

func (a Vault) openConnections(isMissing bool) (domain.Vault, error) {
	v, err := a.open()
	if err != nil {
		return domain.Vault{}, err
	}
	if isMissing {
		return domain.Vault{}, ErrNoConnections
	}
	return v, nil
}

func (a Vault) GetConnection(ctx context.Context, first, second, kind string) (domain.Connection, error) {
	v, err := a.openConnections(a.Connections.Reader == nil)
	if err != nil {
		return domain.Connection{}, err
	}
	return a.Connections.Reader.GetConnection(ctx, v, first, second, kind)
}

func (a Vault) SaveConnection(ctx context.Context, c domain.Connection) error {
	v, err := a.openConnections(a.Connections.Writer == nil)
	if err != nil {
		return err
	}
	return a.notifyAfterWrite(a.Connections.Writer.SaveConnection(ctx, v, c))
}

func (a Vault) RemoveSide(ctx context.Context, from, to, kind string) error {
	v, err := a.openConnections(a.Connections.Writer == nil)
	if err != nil {
		return err
	}
	return a.notifyAfterWrite(a.Connections.Writer.RemoveSide(ctx, v, from, to, kind))
}

func (a Vault) RemoveConnection(ctx context.Context, first, second, kind string) error {
	v, err := a.openConnections(a.Connections.Writer == nil)
	if err != nil {
		return err
	}
	return a.notifyAfterWrite(a.Connections.Writer.RemoveConnection(ctx, v, first, second, kind))
}

func (a Vault) Join(ctx context.Context, from, to string, side domain.Side) error {
	v, err := a.openConnections(a.Connections.Joiner == nil)
	if err != nil {
		return err
	}
	return a.notifyAfterWrite(a.Connections.Joiner.Join(ctx, v, from, to, side))
}

func (a Vault) CreateInRelation(ctx context.Context, in domain.RelatedNote) (string, error) {
	v, err := a.openConnections(a.Connections.Creator == nil)
	if err != nil {
		return "", err
	}
	path, err := a.Connections.Creator.CreateInRelation(ctx, v, in)
	if path != "" {
		a.notifyFileWritten()
	}
	return path, err
}

func (a Vault) notifyAfterWrite(err error) error {
	if err == nil || errors.Is(err, edit.ErrIndexOutdated) {
		a.notifyFileWritten()
	}
	return err
}
