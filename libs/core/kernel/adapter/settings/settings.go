// Package settings reads and writes the installation's settings file.
//
// The file is JSON, in the desktop's configuration folder. Every user-changeable
// setting is a section of it, so all settings are in one file. It is named after
// the application, so its purpose is clear wherever it is displayed or copied.
// The list of vaults is not stored here: the application rewrites it when a vault
// is added, and a file the application rewrites is unsuitable for manually edited
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
// whose fields are the sections, in file order.
// An adapter's configuration is that adapter's section; the document decides its place.
// A document starts with the defaults of an unconfigured installation, so a
// file that omits or half-names a section loads complete.
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
	SyncTitleAndFilename *bool `json:"syncTitleAndFilename"`
}

// Sync is whether a note's title and its filename are kept as one name. A
// section naming nothing keeps them one name.
func (n Titles) Sync() bool {
	return n.SyncTitleAndFilename == nil || *n.SyncTitleAndFilename
}

// DefaultVersion is the shape a file this build writes is written at.
const DefaultVersion = 1

// DefaultAppearance, DefaultTitles and DefaultPerformance are the values of an unconfigured
// installation: the defaults the declaration of the shell gives.
func DefaultAppearance() Appearance {
	return Appearance{
		InterfaceScale: getAppNumber(keyInterfaceScale),
		TextScale:      getAppNumber(keyTextScale),
		Mode:           getAppString(keyMode),
		Theme:          getAppString(keyTheme),
	}
}

func DefaultTitles() Titles {
	isOn := getAppBool(keySyncNaming)
	return Titles{SyncTitleAndFilename: &isOn}
}

func DefaultPerformance() Performance {
	return Performance{Profile: getAppString(keyProfile)}
}

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
func (f *Files) OpenAt(path string, into Document) error {
	if f == nil {
		return ErrNoFiles
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		// An unconfigured installation is written with its effective values, including
		// the size the desktop requests. A machine that cannot write the file runs with
		// the same settings.
		into.GetAppearance().InterfaceScale = getDesktopScale()
		if err := f.write(path, into); err != nil && !isUnwritable(err) {
			warn(into, "%s: cannot write the file: %v; the default settings apply and are not saved", path, err)
		}
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, into); err != nil {
		return fmt.Errorf("%s: %s: %w", path, describeErrorLocation(err), err)
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
	return nil
}

func warn(into Document, response string, about ...any) {
	into.AddNotice(fmt.Sprintf(response, about...))
}

func (f *Files) write(path string, target Document) error {
	raw, err := json.MarshalIndent(target, "", "  ")
	if err != nil {
		return err
	}
	return f.run(path, func(path string) error {
		return writeFileAtomically(path, append(raw, '\n'))
	})
}

// Adjacent returns the path of name in the folder of the registry file, and false while no registry path is
// set.
func Adjacent(registryPath, name string) (string, bool) {
	if registryPath == "" {
		return "", false
	}
	return filepath.Join(filepath.Dir(registryPath), name), true
}

// LocateFile returns the settings file: settingsPath where one is set, else numen.json beside the registry,
// else the file in the platform's own location.
func LocateFile(settingsPath, registryPath string) (string, error) {
	if settingsPath != "" {
		return settingsPath, nil
	}
	if file, isAdjacent := Adjacent(registryPath, "numen.json"); isAdjacent {
		return file, nil
	}
	return Path()
}

// LocateThemes returns the folder of the user's themes: themesPath where one is set, else the themes folder
// beside the registry, and false where neither is set and the platform's own folder holds them.
func LocateThemes(themesPath, registryPath string) (string, bool) {
	if themesPath != "" {
		return themesPath, true
	}
	return Adjacent(registryPath, "themes")
}
