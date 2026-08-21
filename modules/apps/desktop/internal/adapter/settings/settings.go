// Package settings is what a person configures about this installation.
//
// One file, JSON, at `~/.config/numen/numen.json`. Everything a person may want
// to change is a section of it, so finding a setting is finding one file. It is
// named after the application, so the name says what it configures wherever it
// is read out or copied to.
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
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/agent"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/ocr/onnx"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/proofreading"
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
}

// Appearance is how the window is drawn.
type Appearance struct {
	// Zoom is how large everything is drawn, 1 being as designed. Zero asks the
	// desktop instead.
	Zoom float64 `json:"zoom"`
}

// Indexing is how a vault is made searchable.
type Indexing struct {
	// Embedding is which model turns text into vectors, and how it is reached.
	Embedding embed.Config `json:"embedding"`

	// Recognition is how a scanned document is read when a person asks for it.
	// Nothing here runs on its own.
	Recognition onnx.Config `json:"recognition"`

	// Proofreading is what puts a reading right. Naming nothing here is naming
	// no proofreader, and a reading is used as it was read.
	Proofreading proofreading.Config `json:"proofreading"`
}

// Defaults are what an installation nobody has configured does.
func Defaults() Config {
	return Config{
		V: 1,
		Indexing: Indexing{
			Embedding:    embed.Defaults(),
			Recognition:  onnx.Defaults(),
			Proofreading: proofreading.Defaults(),
		},
		Agent: agent.Defaults(),
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
	return cfg, nil
}
