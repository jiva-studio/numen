package versionfile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/textport"
)

// FileName is the file of the core's folder the versions are kept in.
const FileName = "text-versions.json"

// Folder is the core's folder of one vault's store. A file that is not there
// is fs.ErrNotExist; a write replaces the file whole.
type Folder interface {
	Read(ctx context.Context, name string) ([]byte, error)
	Write(ctx context.Context, name string, content []byte) error
}

// Repositories keeps each vault's versions in the core folder its opener gives.
// Every write to one vault's file is a read, a change and a write under one lock.
type Repositories struct {
	open func(v domain.Vault) (Folder, error)
	mu   sync.Mutex
}

// NewRepositories keeps the versions in the folders open gives.
func NewRepositories(open func(v domain.Vault) (Folder, error)) *Repositories {
	return &Repositories{open: open}
}

// Open is the versions of one vault.
func (r *Repositories) Open(v domain.Vault) (textport.VersionRepository, error) {
	folder, err := r.open(v)
	if err != nil {
		return nil, err
	}
	return repository{owner: r, folder: folder}, nil
}

type repository struct {
	owner  *Repositories
	folder Folder
}

// record is one version as the file keeps it.
type record struct {
	Path       string `json:"path"`
	PluginID   string `json:"pluginId"`
	Key        string `json:"key"`
	Input      string `json:"input,omitempty"`
	Format     string `json:"format"`
	InputHash  string `json:"inputHash"`
	Hash       string `json:"hash"`
	CreateTime string `json:"createTime"`
}

type file struct {
	Versions []record `json:"versions"`
}

func (r repository) GetVersions(ctx context.Context, path string) ([]domain.TextVersion, error) {
	all, err := r.ListVersions(ctx)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(all, func(v domain.TextVersion) bool { return v.Path != path }), nil
}

func (r repository) ListVersions(ctx context.Context) ([]domain.TextVersion, error) {
	r.owner.mu.Lock()
	defer r.owner.mu.Unlock()
	return r.read(ctx)
}

func (r repository) SaveVersion(ctx context.Context, version domain.TextVersion) error {
	r.owner.mu.Lock()
	defer r.owner.mu.Unlock()
	all, err := r.read(ctx)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(all, func(v domain.TextVersion) bool { return isSame(v, version.Path, version.GetID()) })
	if i >= 0 {
		all[i] = version
	} else {
		all = append(all, version)
	}
	return r.write(ctx, all)
}

func (r repository) RemoveVersion(ctx context.Context, path string, id domain.TextVersionID) error {
	r.owner.mu.Lock()
	defer r.owner.mu.Unlock()
	all, err := r.read(ctx)
	if err != nil {
		return err
	}
	kept := slices.DeleteFunc(all, func(v domain.TextVersion) bool { return isSame(v, path, id) })
	return r.write(ctx, kept)
}

func isSame(v domain.TextVersion, path string, id domain.TextVersionID) bool {
	return v.Path == path && v.GetID() == id
}

// read is every version the file holds. A file that is not there holds none.
func (r repository) read(ctx context.Context) ([]domain.TextVersion, error) {
	raw, err := r.folder.Read(ctx, FileName)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", FileName, err)
	}
	var stored file
	if err := json.Unmarshal(raw, &stored); err != nil {
		return nil, fmt.Errorf("read %s: %w", FileName, err)
	}
	out := make([]domain.TextVersion, 0, len(stored.Versions))
	for _, one := range stored.Versions {
		created, err := time.Parse(time.RFC3339Nano, one.CreateTime)
		if err != nil {
			return nil, fmt.Errorf("read %s: version %s/%s of %s: %w", FileName, one.PluginID, one.Key, one.Path, err)
		}
		out = append(out, domain.TextVersion{
			Path: one.Path, PluginID: one.PluginID, Key: one.Key, Input: domain.TextVersionID(one.Input),
			Format: domain.TextFormat(one.Format), InputHash: one.InputHash, Hash: one.Hash, CreateTime: created,
		})
	}
	return out, nil
}

func (r repository) write(ctx context.Context, versions []domain.TextVersion) error {
	stored := file{Versions: make([]record, 0, len(versions))}
	for _, v := range versions {
		stored.Versions = append(stored.Versions, record{
			Path: v.Path, PluginID: v.PluginID, Key: v.Key, Input: string(v.Input), Format: string(v.Format),
			InputHash: v.InputHash, Hash: v.Hash, CreateTime: v.CreateTime.UTC().Format(time.RFC3339Nano),
		})
	}
	raw, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	if err := r.folder.Write(ctx, FileName, raw); err != nil {
		return fmt.Errorf("write %s: %w", FileName, err)
	}
	return nil
}
