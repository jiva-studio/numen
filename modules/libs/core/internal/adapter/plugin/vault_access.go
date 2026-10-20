package plugin

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// ErrNoVault is a call for the vault the window is showing when it shows none.
var ErrNoVault = errors.New("the host is showing no vault")

// Vault is port.PluginVault over the ports of the installation.
type Vault struct {
	// Current is the vault the window is showing.
	Current func() (domain.Vault, bool)
	Readers port.VaultReaders
	Writers port.VaultWriters
	Notes   port.NoteQueries
	Links   port.LinkQueries
	// Problems answers what parsing turned up. Without it nothing is.
	Problems port.ProblemQueries
	Data     port.PluginDataStores
	Cache    port.PluginCache
	Watcher  port.VaultWatcher
	// Index brings the index level with the paths a plugin changed. Without it nothing is done.
	Index func(ctx context.Context, v domain.Vault, paths []string) error
}

func (a Vault) GetVault() (domain.Vault, bool) {
	if a.Current == nil {
		return domain.Vault{}, false
	}
	return a.Current()
}

func (a Vault) open() (domain.Vault, error) {
	v, showing := a.GetVault()
	if !showing {
		return domain.Vault{}, ErrNoVault
	}
	return v, nil
}

func (a Vault) openReader() (port.VaultReader, domain.Vault, error) {
	v, err := a.open()
	if err != nil {
		return nil, domain.Vault{}, err
	}
	reader, err := a.Readers.Open(v)
	return reader, v, err
}

func (a Vault) openWriter() (port.VaultWriter, domain.Vault, error) {
	v, err := a.open()
	if err != nil {
		return nil, domain.Vault{}, err
	}
	writer, err := a.Writers.Open(v)
	return writer, v, err
}

func (a Vault) GetFingerprint(ctx context.Context, path string) (domain.Fingerprint, error) {
	reader, _, err := a.openReader()
	if err != nil {
		return domain.Fingerprint{}, err
	}
	return reader.GetFingerprint(ctx, path)
}

func (a Vault) Read(ctx context.Context, path string) ([]byte, domain.Fingerprint, error) {
	reader, _, err := a.openReader()
	if err != nil {
		return nil, domain.Fingerprint{}, err
	}
	at, err := reader.GetFingerprint(ctx, path)
	if err != nil {
		return nil, domain.Fingerprint{}, err
	}
	content, err := reader.Read(ctx, path)
	if err != nil {
		return nil, domain.Fingerprint{}, err
	}
	return content, at, nil
}

func (a Vault) Write(
	ctx context.Context, path string, content []byte, baseline domain.Fingerprint,
) (domain.Fingerprint, error) {
	writer, _, err := a.openWriter()
	if err != nil {
		return domain.Fingerprint{}, err
	}
	return writer.Write(ctx, path, content, baseline)
}

func (a Vault) Create(ctx context.Context, path string, content []byte) error {
	writer, _, err := a.openWriter()
	if err != nil {
		return err
	}
	return writer.Create(ctx, path, content)
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
	if err != nil || a.Index == nil {
		return err
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
	ctx context.Context, from string, written []string,
) (map[string]domain.ResolvedLink, error) {
	v, err := a.open()
	if err != nil {
		return nil, err
	}
	return a.Links.ResolveLinks(ctx, v.ID, from, written)
}

func (a Vault) GetParseProblems(ctx context.Context) ([]domain.VaultProblem, error) {
	v, err := a.open()
	if err != nil || a.Problems == nil {
		return nil, err
	}
	return a.Problems.GetParseProblems(ctx, v.ID)
}

func (a Vault) OpenData(folder string) (port.PluginDataStore, error) {
	v, err := a.open()
	if err != nil {
		return nil, err
	}
	return a.Data.Open(v, folder)
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
		return nil, nil, fmt.Errorf("watch %s: %w", v.ID, port.ErrUnavailable)
	}
	return a.Watcher.WatchVault(ctx, v)
}
