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
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/recognition"
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

	// Review is what a day of review is, on this person's clock. How a deck is
	// scheduled is in the vault, in the preset it points at.
	Review Review `json:"review"`

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

	// HangPartsUnderANode is whether a node in the plex hangs the headings of
	// its note under the box. A file leaving it out hangs them, and a file
	// naming false leaves the box alone.
	HangPartsUnderANode *bool `json:"hang_parts_under_a_node"`

	// PartsUnderANode is how many of those headings stand under a node at once,
	// the rest being wound to. A number outside PartsUnderANodeBounds is
	// refused, and a file naming none stands DefaultParts of them.
	PartsUnderANode int `json:"parts_under_a_node"`
}

// Hangs is whether a node hangs the headings of its note under it. A section
// naming nothing hangs them.
func (a Appearance) Hangs() bool {
	return a.HangPartsUnderANode == nil || *a.HangPartsUnderANode
}

// The modes a colour pair is read by.
const (
	ModeSystem = "system"
	ModeLight  = "light"
	ModeDark   = "dark"
)

// AsDesigned is the multiplier that draws everything the size it was drawn at.
const AsDesigned = 1

// DefaultParts is how many headings stand under a node where the file names no
// number.
const DefaultParts = 6

// Bounds is how far a multiplier goes, at each end.
type Bounds struct{ Least, Most float64 }

// How far each of the two sizes goes, and how many headings a node may hang.
//
// The interface holds while the smallest control it draws is a target a
// pointer finds, and while a window 1280 across still stands its panes side by
// side. Text holds while the smallest of it is still read, and while a line of
// typing still fits the row it is typed in. A node hangs at least one heading,
// and twelve of them reach the foot of a window the plex is drawn in.
var (
	InterfaceScaleBounds  = Bounds{Least: 0.8, Most: 2}
	TextScaleBounds       = Bounds{Least: 0.8, Most: 1.75}
	PartsUnderANodeBounds = Bounds{Least: 1, Most: 12}
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

// Check is what is wrong with the two sizes and the count, and nothing where
// each is a number its setting takes.
func (a Appearance) Check() error {
	if err := InterfaceScaleBounds.Check("appearance.interface_scale", a.InterfaceScale); err != nil {
		return err
	}
	if err := TextScaleBounds.Check("appearance.text_scale", a.TextScale); err != nil {
		return err
	}
	return PartsUnderANodeBounds.Check("appearance.parts_under_a_node", float64(a.PartsUnderANode))
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

// Sync is whether a note's title and its filename are kept as one name. A
// section naming nothing keeps them one name.
func (n Naming) Sync() bool {
	return n.SyncTitleAndFilename == nil || *n.SyncTitleAndFilename
}

// Review is what a day of review is, on this person's clock.
type Review struct {
	// DayStarts is the hour a day of review begins at, on the clock on the
	// wall, written as hours and minutes. An answer given before it is written
	// into the day before.
	DayStarts string `json:"day_starts"`
}

// LatestDayStarts is how far past midnight a day may be made to begin.
const LatestDayStarts = 12 * time.Hour

// ClockFormat is how an hour of the day is written.
const ClockFormat = "15:04"

// Starts is how long past midnight a day of review begins, and whether the file
// said something that is not an hour of the day.
func (r Review) Starts() (time.Duration, bool) {
	written := strings.TrimSpace(r.DayStarts)
	if written == "" {
		return DefaultStarts(), true
	}
	at, err := time.Parse(ClockFormat, written)
	if err != nil {
		return DefaultStarts(), false
	}
	starts := time.Duration(at.Hour())*time.Hour + time.Duration(at.Minute())*time.Minute
	if starts > LatestDayStarts {
		return DefaultStarts(), false
	}
	return starts, true
}

// DefaultStarts is when a day of review begins where the file says nothing. An
// answer given before it finishes the evening it belongs to.
func DefaultStarts() time.Duration { return flashcards.DayStarts }

// Starting is the hour a day of review is to begin at, as it goes into the
// file. An hour past LatestDayStarts, anything that is not an hour of the
// clock, and no hour at all, are flashcards.ErrNotAnHour. It reads and writes
// no file.
func Starting(written string) (string, error) {
	starts, hour := Review{DayStarts: written}.Starts()
	if !hour || strings.TrimSpace(written) == "" {
		return "", fmt.Errorf("%w, 00:00 to %s: %q",
			flashcards.ErrNotAnHour, flashcards.Clock(LatestDayStarts), written)
	}
	return flashcards.Clock(starts), nil
}

// Sync is whether a note's title and its filename are kept as one name.
func (c Config) Sync() bool { return c.Naming.Sync() }

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
		V: 1,
		Appearance: Appearance{
			InterfaceScale:      AsDesigned,
			TextScale:           AsDesigned,
			Mode:                ModeSystem,
			Theme:               DefaultTheme,
			HangPartsUnderANode: on(),
			PartsUnderANode:     DefaultParts,
		},
		Indexing: Indexing{
			Embedding:    embed.Defaults(),
			Recognition:  recognition.Defaults(),
			Proofreading: proofreading.Defaults(),
		},
		Agent:  agent.Defaults(),
		Naming: Naming{SyncTitleAndFilename: on()},
		Review: Review{DayStarts: flashcards.Clock(DefaultStarts())},
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
	if _, hour := cfg.Review.Starts(); !hour {
		cfg.say("review.day_starts is an hour of the day, 00:00 to %s, and %s stands",
			flashcards.Clock(LatestDayStarts), flashcards.Clock(DefaultStarts()))
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
