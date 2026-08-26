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
	"strconv"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/agent"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/recognition"
)

// Config is this installation's settings, in sections named for what they are
// about. A person looking for a setting looks for the part of the application it
// belongs to.
type Config struct {
	// V is the shape of the file. Nothing reads it yet, and it is written so that
	// the day a section changes shape there is something to tell the two apart.
	V int `json:"v"`

	// Appearance is how the window is drawn.
	Appearance Appearance `json:"appearance"`

	// Indexing is how a vault is made searchable.
	Indexing Indexing `json:"indexing"`

	// Agent is which agent answers in the panel, and what it may reach.
	Agent agent.Config `json:"agent"`

	// Naming is how a note's title and the name of its file are held together.
	Naming Naming `json:"naming"`

	// Said is what reading the file leaves a person something to do about: a
	// number written where a setting does not go that far. Each is one line of
	// a band, which gives a line about sixty characters, and whoever read the
	// settings puts it where the person is.
	Said []string `json:"-"`
}

// Appearance is how the window is drawn.
type Appearance struct {
	// InterfaceScale is how large the window is drawn: its chrome, its controls,
	// the spacing between them and the type in them. A number outside
	// InterfaceScaleBounds is refused, and a file naming no size at all is drawn
	// at what the desktop asks for.
	InterfaceScale float64 `json:"interface_scale"`

	// TextScale is how large the text a person reads is set: a note, a book, an
	// answer, the editor. A number outside TextScaleBounds is refused.
	TextScale float64 `json:"text_scale"`

	// Mode is which half of a colour pair the window takes: ModeSystem,
	// ModeLight or ModeDark. A theme that pins the two halves itself leaves
	// this nothing to choose.
	Mode string `json:"mode"`

	// Theme is the stylesheet the window wears, named by the shelf it came off
	// and its filename: `preset:dracula` ships here, `mine:dracula` is the
	// person's file.
	Theme string `json:"theme"`
}

// The modes a colour pair is read by.
const (
	ModeSystem = "system"
	ModeLight  = "light"
	ModeDark   = "dark"
)

// AsDesigned is the multiplier that draws everything the size it was drawn at.
const AsDesigned = 1

// Bounds is how far a multiplier goes, at each end.
type Bounds struct{ Least, Most float64 }

// How far each of the two goes.
//
// The interface holds while the smallest control it draws is a target a
// pointer finds, and while a window 1280 across still stands its panes side by
// side. Text holds while the smallest of it is still read, and while a line of
// typing still fits the row it is typed in.
var (
	InterfaceScaleBounds = Bounds{Least: 0.8, Most: 2}
	TextScaleBounds      = Bounds{Least: 0.8, Most: 1.75}
)

// Holds is whether a number is one the setting takes.
func (b Bounds) Holds(value float64) bool {
	return value >= b.Least && value <= b.Most
}

// Check hands back what is wrong with a number the setting does not take, and
// nothing for one it does. at is where the number sits in the file.
func (b Bounds) Check(at string, value float64) error {
	if b.Holds(value) {
		return nil
	}
	return &Outside{At: at, Value: value, Bounds: b}
}

// Outside is a number a setting does not take, and how far that setting goes.
// The number is left as the person wrote it and nothing is drawn at it.
type Outside struct {
	// At is where the number sits in the file: `appearance.text_scale`.
	At    string
	Value float64
	Bounds
}

func (o *Outside) Error() string {
	return fmt.Sprintf("%s is %v, and goes from %v to %v", o.At, o.Value, o.Least, o.Most)
}

// Check is what is wrong with the two sizes, and nothing where each is a
// number its setting takes.
func (a Appearance) Check() error {
	if err := InterfaceScaleBounds.Check("appearance.interface_scale", a.InterfaceScale); err != nil {
		return err
	}
	return TextScaleBounds.Check("appearance.text_scale", a.TextScale)
}

// DefaultTheme is this product's own palette, which is what an installation
// nobody has dressed wears.
const DefaultTheme = "preset:numen"

// Indexing is how a vault is made searchable.
type Indexing struct {
	// Embedding is which model turns text into vectors, and how it is reached.
	Embedding embed.Config `json:"embedding"`

	// Recognition is how a scanned document is read when a person asks for it.
	// Nothing here runs on its own.
	Recognition recognition.Config `json:"recognition"`

	// Proofreading is what puts a reading right. Naming nothing here is naming
	// no proofreader, and a reading is used as it was read.
	Proofreading proofreading.Config `json:"proofreading"`
}

// Naming is how a note's title and the name of its file are held together.
type Naming struct {
	// SyncTitleAndFilename is whether renaming either of the two brings the
	// other into line. A file leaving it out keeps them one name, and a file
	// naming false is what tells them apart.
	SyncTitleAndFilename *bool `json:"sync_title_and_filename"`
}

// Sync is whether a note's title and its filename are kept as one name.
func (c Config) Sync() bool {
	return c.Naming.SyncTitleAndFilename == nil || *c.Naming.SyncTitleAndFilename
}

// on is a setting turned on.
func on() *bool {
	set := true
	return &set
}

// Defaults are what an installation nobody has configured does.
func Defaults() Config {
	return Config{
		V: 1,
		Appearance: Appearance{
			InterfaceScale: AsDesigned,
			TextScale:      AsDesigned,
			Mode:           ModeSystem,
			Theme:          DefaultTheme,
		},
		Indexing: Indexing{
			Embedding:    embed.Defaults(),
			Recognition:  recognition.Defaults(),
			Proofreading: proofreading.Defaults(),
		},
		Agent:  agent.Defaults(),
		Naming: Naming{SyncTitleAndFilename: on()},
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
	if cfg.V == 0 {
		cfg.V = 1
	}
	cfg.carrying(path, raw)
	if err := cfg.Appearance.Check(); err != nil {
		return Config{}, err
	}
	return cfg, nil
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

// fromDesktop is how large the interface is drawn where the file names no size.
//
// A screen says how many pixels it has and not how large they are, so the
// desktop is asked: GDK_DPI_SCALE is what a person told their session text
// should be scaled by, and the interface is drawn to match. A number outside
// InterfaceScaleBounds is not one the setting is seeded with.
func fromDesktop() float64 {
	scale, err := strconv.ParseFloat(os.Getenv("GDK_DPI_SCALE"), 64)
	if err != nil || !InterfaceScaleBounds.Holds(scale) {
		return AsDesigned
	}
	return scale
}

func (c *Config) say(said string, about ...any) {
	c.Said = append(c.Said, fmt.Sprintf(said, about...))
}

// write puts the settings where they are read from. The key is not among what
// is written: rewriting the file is not how one is set.
func write(path string, cfg Config) error {
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o600)
}
