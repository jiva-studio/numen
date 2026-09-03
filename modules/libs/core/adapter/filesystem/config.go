package filesystem

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/ulid"
)

// DefaultServiceDir is the one folder the application writes into a vault.
const DefaultServiceDir = ".numen"

const configName = "config.json"

// Config is what a vault knows about itself: an identity, a format version,
// and what the person has said not to look at. Nothing that could be
// recomputed.
type Config struct {
	V  int    `json:"v"`
	ID string `json:"id"`
	// Ignore is written by the person, in the syntax of `.gitignore`. An
	// attachments folder, an export directory, a sync client's scratch space:
	// what is noise is a property of this vault.
	Ignore []string `json:"ignore,omitempty"`
}

// ErrNotAVault is returned when a folder has no identity, which means it has
// never been added to the application.
var ErrNotAVault = errors.New("no vault configuration in this folder")

// configAt is where a folder's identity is written.
func configAt(root, serviceDir string) string {
	if serviceDir == "" {
		serviceDir = DefaultServiceDir
	}
	return filepath.Join(root, serviceDir, configName)
}

// ReadConfig returns the identity the vault carries.
func ReadConfig(root, serviceDir string) (Config, error) {
	raw, err := os.ReadFile(configAt(root, serviceDir))
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, ErrNotAVault
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("vault configuration is unreadable: %w", err)
	}
	if !ulid.Valid(c.ID) {
		return Config{}, fmt.Errorf("vault configuration holds %q: %w", c.ID, ulid.ErrInvalid)
	}
	return c, nil
}

// Initialize gives a folder an identity, which is the moment it becomes a vault.
// This is the one write the application makes into a vault without being asked
// to edit something, and it happens because the user added the vault.
//
// An existing configuration is returned untouched: the identity is what every
// row in the index points at.
func Initialize(root, serviceDir string, now time.Time) (Config, error) {
	if serviceDir == "" {
		serviceDir = DefaultServiceDir
	}
	if c, err := ReadConfig(root, serviceDir); err == nil {
		return c, nil
	} else if !errors.Is(err, ErrNotAVault) {
		return Config{}, err
	}

	id, err := ulid.New(now)
	if err != nil {
		return Config{}, err
	}
	c := Config{V: 1, ID: id}

	dir := filepath.Join(root, serviceDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Config{}, err
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return Config{}, err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(dir, configName), raw, 0o644); err != nil {
		return Config{}, err
	}
	return c, nil
}
