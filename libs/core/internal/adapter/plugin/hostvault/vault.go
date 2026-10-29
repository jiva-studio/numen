package hostvault

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jiva-studio/numen/libs/notes/domain"

	completionport "github.com/jiva-studio/numen/libs/core/usecase/completion/completioncontract"
	noteport "github.com/jiva-studio/numen/libs/core/usecase/note/notecontract"
	problemport "github.com/jiva-studio/numen/libs/core/usecase/note/problemcontract"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	vaultport "github.com/jiva-studio/numen/libs/core/usecase/vault/vaultcontract"
)

var ErrNoVault = errors.New("the host is showing no vault")

// Vault is port.PluginVault over the ports of the installation.
type Vault struct {
	Current func() (domain.Vault, bool)
	Readers vaultport.VaultReaders
	Writers vaultport.VaultWriters
	Notes   noteport.NoteQueries
	Links   noteport.LinkQueries
	// Problems answers what parsing turned up. Without it nothing is.
	Problems problemport.ProblemQueries
	Stores   pluginport.PluginStores
	Cache    pluginport.PluginCache
	Watcher  vaultport.VaultWatcher
	// Vaults lists every vault in the installation, and Find returns the vault with a given identity or
	// fs.ErrNotExist. Without them the installation contains only the vault Current names.
	Vaults func() ([]domain.Vault, error)
	Find   func(id domain.VaultID) (domain.Vault, error)
	// IndexState starts reading a vault into the index when it is not there and returns the index state.
	// Without it every vault is treated as indexed.
	IndexState func(ctx context.Context, v domain.Vault) (pluginport.PluginIndexState, error)
	// Moved signals that something moved in some vault. Without it nothing is signalled.
	Moved func(ctx context.Context) (<-chan struct{}, error)
	// Written is called after a plugin has changed the vault.
	Written func()
	// Index brings the index level with the paths a plugin changed. Without it nothing is done.
	Index func(ctx context.Context, v domain.Vault, paths []string) error
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
	var found domain.Vault
	if a.Find != nil {
		v, err := a.Find(id)
		if err != nil {
			return nil, err
		}
		found = v
	} else if v, showing := a.GetVault(); showing && v.ID == id {
		found = v
	} else {
		return nil, fmt.Errorf("vault %q: %w", id, fs.ErrNotExist)
	}
	linked := a
	linked.Current = func() (domain.Vault, bool) { return found, true }
	return linked, nil
}

func (a Vault) EnsureIndexed(ctx context.Context) (pluginport.PluginIndexState, error) {
	v, err := a.open()
	if err != nil {
		return pluginport.PluginIndexState{}, err
	}
	if a.IndexState == nil {
		return pluginport.PluginIndexState{Status: pluginport.PluginIndexed}, nil
	}
	return a.IndexState(ctx, v)
}

func (a Vault) WatchMoves(ctx context.Context) (<-chan struct{}, error) {
	if a.Moved == nil {
		return nil, fmt.Errorf("watch moves: %w", completionport.ErrUnavailable)
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

func (a Vault) openWriter() (vaultport.VaultWriter, error) {
	v, err := a.open()
	if err != nil {
		return nil, err
	}
	return a.Writers.Open(v)
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

func (a Vault) Write(
	ctx context.Context, path string, content []byte, baseline domain.FileStat,
) (domain.FileStat, error) {
	writer, err := a.openWriter()
	if err != nil {
		return domain.FileStat{}, err
	}
	pos, err := writer.Write(ctx, path, content, baseline)
	if err == nil {
		_ = a.Scan(ctx, []string{path})
	}
	return pos, err
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
	_ = a.Scan(ctx, []string{path})
	return nil
}

func (a Vault) Lock(ctx context.Context) (func(), error) {
	v, err := a.open()
	if err != nil {
		return nil, err
	}
	return a.Writers.Lock(ctx, v)
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
	return a.Notes.GetTypes(ctx, v.ID, paths)
}

func (a Vault) GetNamedPaths(ctx context.Context, name string) ([]string, error) {
	v, err := a.open()
	if err != nil {
		return nil, err
	}
	return a.Notes.GetNamedPaths(ctx, v.ID, name)
}

func (a Vault) GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error) {
	v, err := a.open()
	if err != nil {
		return nil, err
	}
	return a.Notes.GetPathsOfType(ctx, v.ID, noteType)
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
		return nil, nil, fmt.Errorf("watch %s: %w", v.ID, completionport.ErrUnavailable)
	}
	return a.Watcher.WatchVault(ctx, v)
}
