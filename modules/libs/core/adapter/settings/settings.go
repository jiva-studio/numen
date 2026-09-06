// Package settings is what a person configures about this installation.
//
// One file, JSON, in the folder this desktop keeps a person's configuration in.
// Everything a person may want to change is a section of it, so finding a
// setting is finding one file. It is named after the application, so the name
// says what it configures wherever it is read out or copied to.
// The list of vaults is not here: the application writes that itself when a
// vault is added, and a file the application rewrites is no place for something
// typed by hand.
//
// A missing file is an untouched installation. Every field the file leaves out
// keeps what the defaults set, so a file naming one setting is a valid file.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/recognition"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/transcription"
)

// Config is this installation's settings, in sections named for what they are
// about. A person looking for a setting looks for the part of the application it
// belongs to.
type Config struct {
	// Version is the shape of the file. Nothing reads it yet, and it is written
	// so that the day a section changes shape there is something to tell the
	// two apart.
	Version int `json:"v"`

	// Appearance is how the window is drawn.
	Appearance Appearance `json:"appearance"`

	// Indexing is how a vault is made searchable.
	Indexing Indexing `json:"indexing"`

	// Agent is which agent answers in the panel, and what it may reach.
	Agent agent.Config `json:"agent"`

	// Titles is how a note's title and the name of its file are held together.
	Titles Titles `json:"naming"`

	// Review is what a day of review is, on this person's clock. How a deck is
	// scheduled is in the vault, in the preset it points at.
	Review Review `json:"review"`

	// Said is what reading the file leaves a person something to do about: a
	// number written where a setting does not go that far. Each is one line of
	// a band, which gives a line about sixty characters, and whoever read the
	// settings puts it where the person is.
	Said []string `json:"-"`
}

// Titles is how a note's title and the name of its file are held together.
type Titles struct {
	// SyncTitleAndFilename is whether renaming either of the two brings the
	// other into line. A file leaving it out keeps them one name, and a file
	// naming false is what tells them apart.
	SyncTitleAndFilename *bool `json:"sync_title_and_filename"`
}

// Sync is whether a note's title and its filename are kept as one name. A
// section naming nothing keeps them one name.
func (n Titles) Sync() bool {
	return n.SyncTitleAndFilename == nil || *n.SyncTitleAndFilename
}

// Sync is whether a note's title and its filename are kept as one name.
func (c Config) Sync() bool { return c.Titles.Sync() }

// DayStarts is how long past midnight a day of review begins.
func (c Config) DayStarts() time.Duration {
	starts, _ := c.Review.Starts()
	return starts
}

// Hangs is whether a node hangs the headings of its note under it.
func (c Config) Hangs() bool { return c.Appearance.Hangs() }

// Parts is how many of those headings stand under a node at once.
func (c Config) Parts() int { return c.Appearance.PartsUnderANode }

// on is a setting turned on.
func on() *bool {
	set := true
	return &set
}

// Defaults are what an installation nobody has configured does.
func Defaults() Config {
	return Config{
		Version: 1,
		Appearance: Appearance{
			InterfaceScale:      AsDesigned,
			TextScale:           AsDesigned,
			Mode:                ModeSystem,
			Theme:               DefaultTheme,
			HangPartsUnderANode: on(),
			PartsUnderANode:     DefaultParts,
		},
		Indexing: Indexing{
			Embedding:            embed.Defaults(),
			Recognition:          Recognition{Config: recognition.Defaults()},
			Proofreading:         proofreading.Defaults(),
			Transcription:        Transcription{Config: transcription.Defaults()},
			TranscribeRecordings: on(),
		},
		Agent:  agent.Defaults(),
		Titles: Titles{SyncTitleAndFilename: on()},
		Review: Review{DayStarts: review.Clock(DefaultStarts())},
	}
}

// Path is where the settings live.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "numen", "numen.json"), nil
}

// Open reads the settings.
func Open() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}
	return At(path)
}

// At is Open with an explicit path.
func At(path string) (Config, error) {
	cfg := Defaults()
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		// An installation nobody has configured is written down as what it is
		// doing, the size the desktop asks for included. A machine that will
		// not take the file runs on the same settings.
		cfg.Appearance.InterfaceScale = fromDesktop()
		_ = write(path, cfg)
		return cfg, nil
	}
	if err != nil {
		return Config{}, err
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, err
	}
	// The reading above takes a bare null and leaves the defaults standing, so
	// the bytes are asked again whether they are one object.
	if err := object(raw); err != nil {
		return Config{}, err
	}
	// A file naming no version, and one naming a number that is no version at
	// all, are both the shape the sections have always had.
	if cfg.Version < 1 {
		cfg.Version = 1
	}
	cfg.carrying(path, raw)
	if err := cfg.Appearance.Check(); err != nil {
		return Config{}, err
	}
	cfg.wearing()
	if _, hour := cfg.Review.Starts(); !hour {
		cfg.say("review.day_starts is an hour of the day, 00:00 to %s, and %s stands",
			review.Clock(LatestDayStarts), review.Clock(DefaultStarts()))
	}
	return cfg, nil
}

// wearing stands the machine's own choice where the file names a word that is
// no half of a colour pair, and says so. The word is left in the file, where
// the person wrote it and where they will read it again.
func (c *Config) wearing() {
	switch c.Appearance.Mode {
	case ModeSystem, ModeLight, ModeDark:
		return
	}
	c.say("appearance.mode is %s, %s or %s, and %s stands",
		ModeSystem, ModeLight, ModeDark, ModeSystem)
	c.Appearance.Mode = ModeSystem
}

// carrying reads `appearance.zoom` as the setting that replaced it, and gives
// the field that name in the file. Both are how large the window is drawn, so
// the number stands as it was and the window is drawn the size it was.
//
// A file that cannot be written is read this way at every launch, and drawn at
// the same size at every launch.
func (c *Config) carrying(path string, raw []byte) {
	var file struct {
		Appearance struct {
			Zoom           *float64 `json:"zoom"`
			InterfaceScale *float64 `json:"interface_scale"`
		} `json:"appearance"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return
	}
	// A file naming the size is drawn at it, and keeps the names it holds: one
	// section holds one of a name.
	if file.Appearance.InterfaceScale != nil {
		return
	}

	// Zero named no size, and neither did a file that named neither.
	var zoom float64
	if file.Appearance.Zoom != nil {
		zoom = *file.Appearance.Zoom
	}
	switch {
	case zoom <= 0:
		c.Appearance.InterfaceScale = fromDesktop()
	case InterfaceScaleBounds.Holds(zoom):
		c.Appearance.InterfaceScale = zoom
		_ = rename(path, []string{"appearance", "zoom"}, "interface_scale")
	default:
		// One line of a band: the setting the number would be read as, and how
		// far that goes. The number itself stays in the file, where a person
		// wrote it and where they will read it again.
		c.say("appearance.zoom is outside interface_scale, %v to %v",
			InterfaceScaleBounds.Least, InterfaceScaleBounds.Most)
	}
}

func (c *Config) say(said string, about ...any) {
	c.Said = append(c.Said, fmt.Sprintf(said, about...))
}

// write puts the settings where they are read from, beside the file and
// renamed over the top. The key is not among what is written: rewriting the
// file is not how one is set.
func write(path string, cfg Config) error {
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return reaching(path, func(path string) error {
		return replace(path, append(raw, '\n'))
	})
}
