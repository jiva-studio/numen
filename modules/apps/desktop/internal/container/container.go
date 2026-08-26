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

	adapteragent "github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/agent"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/recognition"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/settings"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/trash"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

// Config is what the user may point somewhere else. Empty fields mean the
// platform's own locations.
type Config struct {
	IndexPath    string
	RegistryPath string
	ServiceDir   string
	// SettingsPath is the file a person configures this installation in.
	SettingsPath string
	// ThemesPath is the folder the person's own themes are read from.
	ThemesPath string

	// InterfaceScale is how large the window is drawn and TextScale how large the
	// text a person reads is set, said for this launch alone. Each stands over
	// the settings file until a person chooses that size themselves, and zero is
	// not said.
	InterfaceScale, TextScale float64
	// Extensions are the file extensions treated as notes. Empty means the
	// default, which is markdown alone.
	Extensions []string

	// BookExtensions are the file extensions treated as books. Empty means the
	// default, which is every format a reader takes text out of.
	BookExtensions []string

	// Recognition is how a scanned document is read when a person asks for it.
	Recognition recognition.Config

	// Embedding is the model this run turns text into vectors with. An entry
	// point reads the settings and says what it found, so nothing below one
	// reaches the machine's own file. A zero value names no embedder, and nothing
	// is embedded.
	Embedding embed.Config

	// Proofreading is what puts a reading right. It arrives the way Embedding
	// does, and naming nothing here is naming no proofreader.
	Proofreading proofreading.Config

	// Agent is which agent answers in the panel. It arrives the way Embedding
	// does.
	Agent adapteragent.Config

	// Sync is whether a note's title and its filename are kept as one name. It
	// arrives the way Embedding does.
	Sync note.Sync

	// RebuildIndex reads every file and puts it in the index again, whatever the
	// index remembers about it. Both entry points offer it under one name: a
	// person with a vault restored from an archive is not asked which binary they
	// are holding.
	RebuildIndex bool
}

// Indexing is this configuration carrying what a settings file says about
// making a vault searchable.
//
// Every section of it is carried here, in one place both entry points use. A
// section an entry point leaves behind is a part of the application that does
// nothing and says nothing, since naming no model is how a person turns one
// off.
func (c Config) Indexing(said settings.Indexing) Config {
	c.Embedding = said.Embedding
	c.Recognition = said.Recognition
	c.Proofreading = said.Proofreading
	return c
}

// Settings are what a person has configured this installation to do. An
// installation nobody has configured is written down as what it is doing.
func (c Config) Settings() (settings.Config, error) {
	path, err := c.settingsFile()
	if err != nil {
		return settings.Config{}, err
	}
	return settings.At(path)
}

// settingsFile is the file a person configures this installation in.
func (c Config) settingsFile() (string, error) {
	if c.SettingsPath != "" {
		return c.SettingsPath, nil
	}
	if file, chosen := c.beside("numen.json"); chosen {
		return file, nil
	}
	return settings.Path()
}

// beside is where this installation keeps a file of its own. A registry pointed
// somewhere chosen takes everything else with it, which is what a test and a
// second installation both need.
func (c Config) beside(name string) (string, bool) {
	if c.RegistryPath == "" {
		return "", false
	}
	return filepath.Join(filepath.Dir(c.RegistryPath), name), true
}

// Registry is the list of vaults this installation knows: application state,
// kept with the application.
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

// Trash is the place this machine keeps what a person deleted.
func (c Config) Trash() port.Trash { return trash.New() }

// VaultOptions is how a vault on disk is read: which folder is ours, and which
// files count as notes and as books. The same answer for whatever looks at it.
func (c Config) VaultOptions() filesystem.Options {
	return filesystem.Options{
		ServiceDir:     c.ServiceDir,
		Extensions:     c.Extensions,
		BookExtensions: c.BookExtensions,
	}
}

// DerivedStores opens the shelf the application keeps its own irreplaceable
// files on, inside a vault. It is a third opener beside the readers and the
// writers because it is a third right: reading a person's vault, changing it,
// and keeping something of our own in it are not the same permission.
func (c Config) DerivedStores() port.DerivedStores {
	return filesystem.DerivedStores{Options: c.VaultOptions(), Area: filesystem.OCRDir}
}

// indexPath defaults to the platform cache directory. The index is a cache in
// the strict sense — losing it costs a rebuild and nothing else — so it belongs
// where the system keeps disposable data.
// IndexPathOrDefault is where the index is, whether or not one was named. It is
// what a person is told when the index is what stopped the application.
func (c Config) IndexPathOrDefault() (string, error) { return c.indexPath() }

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
