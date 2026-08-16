// Package container is the composition root: the only place that knows which
// concrete adapter satisfies which port.
//
// Use cases above it work with the interfaces in core/port. That is the point
// of the arrangement: to replace the index or the vault reader, this package
// changes and nothing else does.
package container

import (
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// Config is what the user may point somewhere else. Empty fields mean the
// platform's own locations.
type Config struct {
	IndexPath    string
	RegistryPath string
	ServiceDir   string
	// Extensions are the file extensions treated as notes. Empty means the
	// default, which is markdown alone.
	Extensions []string
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
	return filesystem.Readers{Options: c.VaultOptions()}
}

// VaultWriters opens vaults for changing. It is a separate opener from the
// readers because reading and writing a person's notes are different rights.
func (c Config) VaultWriters() port.VaultWriters {
	return filesystem.Writers{Options: c.VaultOptions()}
}

// VaultWatcher follows vaults for changes the application did not make.
func (c Config) VaultWatcher() port.VaultWatcher {
	return filesystem.Watcher{Options: c.VaultOptions()}
}

// VaultIdentity gives folders their identity.
func (c Config) VaultIdentity() port.VaultIdentity {
	return filesystem.Identity{Options: c.VaultOptions()}
}

// VaultOptions is how a vault on disk is read: which folder is ours, and which
// files count as notes. The same answer for whatever looks at it.
func (c Config) VaultOptions() filesystem.Options {
	return filesystem.Options{ServiceDir: c.ServiceDir, Extensions: c.Extensions}
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
