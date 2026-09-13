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

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// Document is the settings file as whoever composed the installation declares
// it: one object whose fields are the sections, in the order they stand in the
// file. This adapter reads and writes that file and owns the sections below;
// what configures an adapter is that adapter's section, and where each stands
// is the document's to say.
//
// A document arrives holding what an installation nobody has configured does,
// so a file that leaves a section out, and a file that names half of one, both
// come back whole.
type Document interface {
	// GetVersion is the shape the file was written at, GetAppearance how the
	// window is drawn, and GetReview where a day of review begins. Each is read
	// and put right as the file is read.
	GetVersion() *int
	GetAppearance() *Appearance
	GetReview() *Review

	// Say leaves a person one line about what the file holds.
	Say(said string)
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

// turnOn is a setting turned on.
func turnOn() *bool {
	set := true
	return &set
}

// DefaultVersion is the shape a file this build writes is written at.
const DefaultVersion = 1

// DefaultAppearance, DefaultTitles and DefaultReview are what an installation
// nobody has configured draws, names and reviews by.
func DefaultAppearance() Appearance {
	return Appearance{
		InterfaceScale:      AsDesigned,
		TextScale:           AsDesigned,
		Mode:                ModeSystem,
		Theme:               DefaultTheme,
		HangPartsUnderANode: turnOn(),
		PartsUnderANode:     DefaultParts,
	}
}

func DefaultTitles() Titles { return Titles{SyncTitleAndFilename: turnOn()} }

func DefaultReview() Review { return Review{DayStarts: review.Clock(DefaultStarts())} }

// Path is where the settings live.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "numen", "numen.json"), nil
}

// OpenAt reads the file into the document handed in. A file that is not there
// is an untouched installation, and is written down as what it is doing.
func OpenAt(path string, into Document) error {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		// An installation nobody has configured is written down as what it is
		// doing, the size the desktop asks for included. A machine that will
		// not take the file runs on the same settings.
		into.GetAppearance().InterfaceScale = fromDesktop()
		_ = write(path, into)
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, into); err != nil {
		return err
	}
	// The reading above takes a bare null and leaves the defaults standing, so
	// the bytes are asked again whether they are one object.
	if err := object(raw); err != nil {
		return err
	}
	// A file naming no version, and one naming a number that is no version at
	// all, are both the shape the sections have always had.
	if version := into.GetVersion(); *version < DefaultVersion {
		*version = DefaultVersion
	}
	readInterfaceScale(into, path, raw)
	if err := into.GetAppearance().Check(); err != nil {
		return err
	}
	checkMode(into)
	if _, hour := into.GetReview().Starts(); !hour {
		say(into, "review.day_starts is an hour of the day, 00:00 to %s, and %s stands",
			review.Clock(LatestDayStarts), review.Clock(DefaultStarts()))
	}
	return nil
}

// checkMode stands the machine's own choice where the file names a word that is
// no half of a colour pair, and says so. The word is left in the file, where
// the person wrote it and where they will read it again.
func checkMode(into Document) {
	drawn := into.GetAppearance()
	switch drawn.Mode {
	case ModeSystem, ModeLight, ModeDark:
		return
	}
	say(into, "appearance.mode is %s, %s or %s, and %s stands",
		ModeSystem, ModeLight, ModeDark, ModeSystem)
	drawn.Mode = ModeSystem
}

// readInterfaceScale reads `appearance.zoom` as the setting that replaced it,
// and gives the field that name in the file. Both are how large the window is
// drawn, so the number stands as it was and the window is drawn the size it was.
//
// A file that cannot be written is read this way at every launch, and drawn at
// the same size at every launch.
func readInterfaceScale(into Document, path string, raw []byte) {
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
		into.GetAppearance().InterfaceScale = fromDesktop()
	case InterfaceScaleBounds.Contains(zoom):
		into.GetAppearance().InterfaceScale = zoom
		_ = rename(path, []string{"appearance", "zoom"}, "interface_scale")
	default:
		// One line of a band: the setting the number would be read as, and how
		// far that goes. The number itself stays in the file, where a person
		// wrote it and where they will read it again.
		say(into, "appearance.zoom is outside interface_scale, %v to %v",
			InterfaceScaleBounds.Least, InterfaceScaleBounds.Most)
	}
}

func say(into Document, said string, about ...any) {
	into.Say(fmt.Sprintf(said, about...))
}

// write puts the settings where they are read from, beside the file and
// renamed over the top. The key is not among what is written: rewriting the
// file is not how one is set.
func write(path string, held Document) error {
	raw, err := json.MarshalIndent(held, "", "  ")
	if err != nil {
		return err
	}
	return runOnFile(path, func(path string) error {
		return replace(path, append(raw, '\n'))
	})
}
