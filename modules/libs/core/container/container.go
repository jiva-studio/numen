// Package container is the composition root: the only place that knows which
// concrete adapter satisfies which port.
//
// Use cases above it work with the interfaces in core/port. That is the point
// of the arrangement: to replace the index or the vault reader, this package
// changes and nothing else does.
package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	adapteragent "github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/recognition"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/transcription"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/trash"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
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
	// SchedulesPath is the folder the schedules worked out from a vault's
	// answers are cached in. It is a cache and belongs to the installation, so
	// a test names one of its own rather than filling the machine's.
	SchedulesPath string

	// InterfaceScale is how large the window is drawn and TextScale how large the
	// text a person reads is set, said for this launch alone. Each stands over
	// the settings file until a person chooses that size themselves, and zero is
	// not said.
	InterfaceScale, TextScale float64

	// BookExtensions are the file extensions treated as books. Empty means the
	// default, which is every format a reader takes text out of.
	BookExtensions []string

	// Recognition is how a scanned document is read when a person asks for it.
	Recognition recognition.Config

	// Transcription is how a recording is listened to.
	Transcription transcription.Config

	// Transcribes is whether a recording the vault holds no transcript for is
	// listened to without anybody asking. A configuration naming nothing leaves
	// it to the hand.
	Transcribes bool

	// TranscribesUnder is how many bytes a recording may run to and still be
	// listened to unasked. A larger one is left for somebody to ask for by
	// name. Zero is no limit.
	TranscribesUnder int64

	// Embedding is the model this run turns text into vectors with. An entry
	// point reads the settings and says what it found, so nothing below one
	// reaches the machine's own file. A zero value names no embedder, and nothing
	// is embedded.
	Embedding embed.Config

	// Proofreading is what puts a reading right. It arrives the way Embedding
	// does, and naming no profile here is naming no proofreader.
	Proofreading proofreading.Config

	// ScanProofreading and SpeechProofreading name the profile each kind of
	// reading is put right at, and say whether that happens without anybody
	// asking.
	ScanProofreading   proofreading.Proofread
	SpeechProofreading proofreading.Proofread

	// AgentProofreader opens a profile that reaches the command line a person
	// already has. The platform supplies it, since core starts no process; an
	// installation that supplies none names no such profile.
	AgentProofreader func(AgentProofreading) (port.Proofreader, error)

	// Agent is which agent answers in the panel. It arrives the way Embedding
	// does.
	Agent adapteragent.Config

	// RebuildIndex reads every file and puts it in the index again, whatever the
	// index remembers about it. Both entry points offer it under one name: a
	// person with a vault restored from an archive is not asked which binary they
	// are holding.
	RebuildIndex bool

	// Trouble is where what is assembled here says what went wrong in work it
	// carries on past. An installation that sets none is told nothing.
	Trouble port.Trouble
}

// trouble says what went wrong to whoever asked to be told.
func (c Config) trouble(err error) {
	if c.Trouble != nil {
		c.Trouble(err)
	}
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
	c.Recognition = said.Recognition.Config
	c.Proofreading = said.Proofreading
	c.ScanProofreading = said.Recognition.Proofread
	c.SpeechProofreading = said.Transcription.Proofread
	c.Transcription = said.Transcription.Config
	c.Transcribes = said.Transcribes()
	c.TranscribesUnder = said.TranscribesUnder()
	return c
}

// Syncing reads, as each rename is made, whether a note's title and its
// filename are kept as one name. A file that cannot be read keeps them one
// name, which is what an installation nobody has configured does.
func (c Config) Syncing() note.Syncing {
	return func() note.SyncTitleAndFilename {
		path, err := c.settingsFile()
		if err != nil {
			return true
		}
		held, err := settings.At(path)
		if err != nil {
			return true
		}
		return note.SyncTitleAndFilename(held.Sync())
	}
}

// Turns writes into the settings whether a note's title and its filename are
// kept as one name. The file is patched as an object, so every key a person
// typed stays where it was.
func (c Config) Turns() func(kept note.SyncTitleAndFilename) error {
	return func(kept note.SyncTitleAndFilename) error {
		path, err := c.settingsFile()
		if err != nil {
			return err
		}
		return settings.Save(path, settings.Setting{
			At: []string{"naming", "sync_title_and_filename"}, Value: bool(kept),
		})
	}
}

// Hanging reads, as the window asks, whether a node hangs the headings of its
// note under it. A file that cannot be read hangs them, which is what an
// installation nobody has configured does.
func (c Config) Hanging() func() bool {
	return func() bool {
		path, err := c.settingsFile()
		if err != nil {
			return true
		}
		held, err := settings.At(path)
		if err != nil {
			return true
		}
		return held.Hangs()
	}
}

// TurnsHanging writes into the settings whether a node hangs the headings of
// its note under it. The file is patched as an object, so every key a person
// typed stays where it was.
func (c Config) TurnsHanging() func(hangs bool) error {
	return func(hangs bool) error {
		path, err := c.settingsFile()
		if err != nil {
			return err
		}
		return settings.Save(path, settings.Setting{
			At: []string{"appearance", "hang_parts_under_a_node"}, Value: hangs,
		})
	}
}

// Parts reads, as the window asks, how many headings stand under a node at
// once. A file that cannot be read stands the default of them.
func (c Config) Parts() func() int {
	return func() int {
		path, err := c.settingsFile()
		if err != nil {
			return settings.DefaultParts
		}
		held, err := settings.At(path)
		if err != nil {
			return settings.DefaultParts
		}
		return held.Parts()
	}
}

// TurnsParts writes into the settings how many headings stand under a node at
// once. A number outside what the setting goes to is refused and the file is
// left as it is.
func (c Config) TurnsParts() func(parts int) error {
	return func(parts int) error {
		if err := settings.PartsUnderANodeBounds.Check(
			"appearance.parts_under_a_node", float64(parts),
		); err != nil {
			return err
		}
		path, err := c.settingsFile()
		if err != nil {
			return err
		}
		return settings.Save(path, settings.Setting{
			At: []string{"appearance", "parts_under_a_node"}, Value: parts,
		})
	}
}

// Reviewing reads, as the window asks, the hour a day of review begins at. A
// file that cannot be read begins the day where an installation nobody has
// configured begins it.
func (c Config) Reviewing() func() string {
	return func() string { return flashcards.Clock(c.DayStarts()) }
}

// TurnsReviewing writes into the settings the hour a day of review begins at.
// An hour the setting does not take is refused and the file is left as it is.
func (c Config) TurnsReviewing() func(starts string) error {
	return func(starts string) error {
		written, err := settings.Starting(starts)
		if err != nil {
			return err
		}
		path, err := c.settingsFile()
		if err != nil {
			return err
		}
		return settings.Save(path, settings.Setting{
			At: []string{"review", "day_starts"}, Value: written,
		})
	}
}

// Configured reads, as the window asks, every setting as JSON and the file it
// stands in.
func (c Config) Configured() func() (string, string, error) {
	return func() (string, string, error) {
		path, err := c.settingsFile()
		if err != nil {
			return "", "", err
		}
		held, err := settings.At(path)
		if err != nil {
			return "", path, err
		}
		written, err := settings.Written(held)
		return written, path, err
	}
}

// ConfiguredFile reads, as the window asks, the settings file as its person
// wrote it, and the file it stands in.
func (c Config) ConfiguredFile() func() (string, string, error) {
	return func() (string, string, error) {
		path, err := c.settingsFile()
		if err != nil {
			return "", "", err
		}
		raw, err := settings.Read(path)
		if err != nil {
			return "", path, err
		}
		return string(raw), path, nil
	}
}

// WritesConfiguredFile replaces the settings file whole, with the bytes as they
// were typed. A file the settings could not be read out of is refused and the
// file is left as it was.
//
// Seen is the file as the window last read it. A file standing at anything else
// is left alone with port.ErrChanged.
func (c Config) WritesConfiguredFile() func(written string, seen *string) error {
	return func(written string, seen *string) error {
		path, err := c.settingsFile()
		if err != nil {
			return err
		}
		return settings.Write(path, []byte(written), seen)
	}
}

// Models reads, as the window asks, the models each setting that names one can
// be set to, and the programs the agent setting can name. The settings are read
// with them, so every row is answered against what is in force; a file that
// cannot be read is answered against the defaults.
func (c Config) Models() func() []port.Model {
	return func() []port.Model {
		held := settings.Defaults()
		if path, err := c.settingsFile(); err == nil {
			if read, err := settings.At(path); err == nil {
				held = read
			}
		}
		return append(settings.Models(held), settings.Agents()...)
	}
}

// TurnsSetting writes settings into the file. The file is patched as an object,
// so every key a person typed stays where it was, and a value the settings
// could not be read out of again is refused before anything is written.
func (c Config) TurnsSetting() func(written []port.Setting) error {
	return func(written []port.Setting) error {
		held := make([]settings.Setting, 0, len(written))
		for _, one := range written {
			var value json.RawMessage
			if err := json.Unmarshal([]byte(one.Value), &value); err != nil {
				return fmt.Errorf("%w: %w", port.ErrNotASetting, err)
			}
			held = append(held, settings.Setting{At: one.At, Value: value})
		}
		path, err := c.settingsFile()
		if err != nil {
			return err
		}
		return settings.Save(path, held...)
	}
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
	return filesystem.VaultReaders{Options: c.VaultOptions()}
}

// VaultWriters opens vaults for changing. It is a separate opener from the
// readers because reading and writing a person's notes are different rights.
func (c Config) VaultWriters() port.VaultWriters {
	return filesystem.VaultWriters{Options: c.VaultOptions()}
}

// VaultWatcher follows vaults for changes the application did not make.
func (c Config) VaultWatcher() port.VaultWatcher {
	return filesystem.Watcher{Options: c.VaultOptions()}
}

// VaultIdentity gives folders their identity.
func (c Config) VaultIdentity() port.VaultIdentity {
	return filesystem.VaultIdentity{Options: c.VaultOptions()}
}

// Trash is the place this machine keeps what a person deleted.
func (c Config) Trash() port.Trash { return trash.New() }

// VaultOptions is how a vault on disk is read: which folder is ours, and which
// files count as books. The same answer for whatever looks at it.
func (c Config) VaultOptions() filesystem.Options {
	return filesystem.Options{
		ServiceDir:     c.ServiceDir,
		BookExtensions: c.BookExtensions,
	}
}

// DerivedStores opens the shelf the application keeps its own irreplaceable
// files on, inside a vault. It is a third opener beside the readers and the
// writers because it is a third right: reading a person's vault, changing it,
// and keeping something of our own in it are not the same permission.
//
// It answers for what a reading wrote and for what a transcription wrote, since
// a use case that places a passage reads both.
func (c Config) DerivedStores() port.DerivedStores {
	return filesystem.DerivedStores{
		Options: c.VaultOptions(),
		Area:    filesystem.OCRDir,
		Areas:   []string{filesystem.SpeechDir},
	}
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
