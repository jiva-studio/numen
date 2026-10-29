// Package settings reads and writes the installation's settings file.
//
// The file is JSON, in the desktop's configuration folder. Every user-changeable
// setting is a section of it, so all settings are in one file. It is named after
// the application, so its purpose is clear wherever it is displayed or copied.
// The list of vaults is not stored here: the application rewrites it when a vault
// is added, and a file the application rewrites is unsuitable for hand-edited
// values.
//
// A missing file is an untouched installation. Every field the file omits keeps
// its default, so a file naming one setting is valid.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Document is the settings file as declared by the composition root: one object
// whose fields are the sections, in file order. This adapter reads and writes
// that file and owns the sections below; an adapter's configuration is that
// adapter's section, and the document determines where each section is placed.
//
// A document is created with the defaults of an unconfigured installation, so a
// file that omits a section, or names half of one, loads complete.
type Document interface {
	// GetVersion is the shape the file was written at, GetAppearance how the
	// window is drawn. Each is read and put right as the file is read.
	GetVersion() *int
	GetAppearance() *Appearance

	// AddNotice records one line of text about the file's contents.
	AddNotice(reply string)
}

// Titles is whether a note's title and its filename are kept in sync.
type Titles struct {
	// SyncTitleAndFilename is whether renaming either the title or the filename
	// updates the other. A file that omits it keeps them identical; a file with false
	// allows them to differ.
	SyncTitleAndFilename *bool `json:"sync_title_and_filename"`
}

// Sync is whether a note's title and its filename are kept as one name. A
// section naming nothing keeps them one name.
func (n Titles) Sync() bool {
	return n.SyncTitleAndFilename == nil || *n.SyncTitleAndFilename
}

// newTrue is a setting turned on.
func newTrue() *bool {
	set := true
	return &set
}

// DefaultVersion is the shape a file this build writes is written at.
const DefaultVersion = 1

// DefaultAppearance and DefaultTitles are the values of an unconfigured installation.
func DefaultAppearance() Appearance {
	return Appearance{
		InterfaceScale: AsDesigned,
		TextScale:      AsDesigned,
		Mode:           ModeSystem,
		Theme:          DefaultTheme,
	}
}

func DefaultTitles() Titles { return Titles{SyncTitleAndFilename: newTrue()} }

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
		// An unconfigured installation is written with its effective values, including
		// the size the desktop requests. A machine that cannot write the file runs with
		// the same settings.
		into.GetAppearance().InterfaceScale = getDesktopScale()
		_ = write(path, into)
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, into); err != nil {
		return err
	}
	// Unmarshal accepts a bare null and leaves the defaults in place, so the bytes
	// are checked again to be a single object.
	if err := checkObject(raw); err != nil {
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
	return nil
}

// checkMode substitutes the machine's own choice where the file names a word that
// is not half of a colour pair, and reports it. The word remains in the file.
func checkMode(into Document) {
	appearance := into.GetAppearance()
	switch appearance.Mode {
	case ModeSystem, ModeLight, ModeDark:
		return
	}
	warn(into, "appearance.mode is %s, %s or %s, and %s stands",
		ModeSystem, ModeLight, ModeDark, ModeSystem)
	appearance.Mode = ModeSystem
}

// readInterfaceScale reads `appearance.zoom` as the setting that replaced it and
// renames the field in the file. Both express the window's display scale, so the
// number and the resulting size are unchanged.
//
// A file that cannot be written is read this way at every launch and yields the
// same size each time.
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
		into.GetAppearance().InterfaceScale = getDesktopScale()
	case InterfaceScaleBounds.Contains(zoom):
		into.GetAppearance().InterfaceScale = zoom
		_ = renameField(path, []string{"appearance", "zoom"}, "interface_scale")
	default:
		// Warns with the setting the number maps to and its bounds. The number remains
		// in the file.
		warn(into, "appearance.zoom is outside interface_scale, %v to %v",
			InterfaceScaleBounds.Min, InterfaceScaleBounds.Max)
	}
}

func warn(into Document, response string, about ...any) {
	into.AddNotice(fmt.Sprintf(response, about...))
}

// write puts the settings where they are read from, beside the file and
// renamed over the top.
func write(path string, target Document) error {
	raw, err := json.MarshalIndent(target, "", "  ")
	if err != nil {
		return err
	}
	return runOnFile(path, func(path string) error {
		return writeFileAtomically(path, append(raw, '\n'))
	})
}
