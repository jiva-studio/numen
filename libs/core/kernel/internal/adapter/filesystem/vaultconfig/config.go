package vaultconfig

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/atomicfile"
	"github.com/jiva-studio/numen/libs/note/ulid"
)

// DefaultServiceDir is where the application keeps a vault's own state. The
// other folder it writes into a vault is TrashDir.
const DefaultServiceDir = ".numen"

const configName = "config.json"

// Config is what a vault records about itself: an identity, a format version,
// and what the user has excluded. Nothing that could be recomputed.
type Config struct {
	Version int      `json:"v"`
	ID      string   `json:"id"`
	Ignore  []string `json:"ignore,omitempty"`
}

// ErrNotAVault is returned when a folder has no identity, which means it has
// never been added to the application.
var ErrNotAVault = errors.New("no vault configuration in this folder")

// ConfigAt is where a folder's identity is written.
func ConfigAt(root, serviceDir string) string {
	if serviceDir == "" {
		serviceDir = DefaultServiceDir
	}
	return filepath.Join(root, serviceDir, configName)
}

// ReadConfig returns the identity the vault carries.
func ReadConfig(root, serviceDir string) (Config, error) {
	raw, err := os.ReadFile(ConfigAt(root, serviceDir))
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
// It is the one write the application makes into a vault without a request to
// edit something; it happens because the user added the vault.
// An existing configuration is returned untouched: every index row points at
// the identity.
func Initialize(root, serviceDir string, now time.Time) (Config, error) {
	if serviceDir == "" {
		serviceDir = DefaultServiceDir
	}
	if c, err := ReadConfig(root, serviceDir); err == nil {
		return c, nil
	} else if !errors.Is(err, ErrNotAVault) {
		return Config{}, err
	}

	id, err := ulid.NewGenerator(rand.Reader).NewID(now)
	if err != nil {
		return Config{}, err
	}
	c := Config{Version: 1, ID: id}

	dir := filepath.Join(root, serviceDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Config{}, err
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return Config{}, err
	}
	raw = append(raw, '\n')
	if err := atomicfile.WriteFile(context.Background(), filepath.Join(dir, configName), bytes.NewReader(raw), 0o644); err != nil {
		return Config{}, err
	}
	return c, nil
}
