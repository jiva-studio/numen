// Package container is the composition root: the only place that knows which
// concrete adapter satisfies which port.
//
// Use cases above it work with the interfaces in core/port. That is the point
// of the arrangement: to replace the index or the vault reader, this package
// changes and nothing else does.
package container

import (
	"context"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/index"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// Config is what the user may point somewhere else. Empty fields mean the
// platform's own locations.
type Config struct {
	IndexPath    string
	RegistryPath string
	ServiceDir   string
}

// Registry is the list of vaults this installation knows: application state,
// kept with the application rather than in any vault.
func (c Config) Registry() (port.VaultRegistry, error) {
	if c.RegistryPath != "" {
		return appstate.At(c.RegistryPath), nil
	}
	return appstate.Open()
}

// VaultReaders opens vaults for reading.
func (c Config) VaultReaders() port.VaultReaders {
	return filesystem.Readers{ServiceDir: c.serviceDir()}
}

// VaultIdentity gives folders their identity.
func (c Config) VaultIdentity() port.VaultIdentity {
	return filesystem.Identity{ServiceDir: c.serviceDir()}
}

// Index opens the cache. Closing it belongs to the caller, which is what knows
// when it is finished.
func (c Config) Index(ctx context.Context) (*index.DB, error) {
	path, err := c.indexPath()
	if err != nil {
		return nil, err
	}
	return index.Open(ctx, path)
}

func (c Config) serviceDir() string {
	if c.ServiceDir == "" {
		return filesystem.DefaultServiceDir
	}
	return c.ServiceDir
}

// indexPath defaults to the platform cache directory. The index is a cache in
// the strict sense — losing it costs a rebuild and nothing else — so it belongs
// where the system keeps disposable data.
func (c Config) indexPath() (string, error) {
	if c.IndexPath != "" {
		if err := os.MkdirAll(filepath.Dir(c.IndexPath), 0o755); err != nil {
			return "", err
		}
		return c.IndexPath, nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "numen")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "index.db"), nil
}
