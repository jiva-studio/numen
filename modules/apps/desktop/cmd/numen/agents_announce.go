//go:build !nomcp

package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
)

// announcement is where agents may reach this vault and what they must present.
//
// It is written where the application keeps its own state rather than in a
// vault: it belongs to this installation and is worth nothing on another
// machine. Configuring an agent is then reading a file rather than watching a
// log.
type announcement struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

// announce writes the file, and hands back what removes it. A file left behind
// points the next agent at a dead port with a live-looking token.
func announce(cfg container.Config, url, token string) (func(), error) {
	path, err := announcementPath(cfg)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	raw, err := json.MarshalIndent(announcement{URL: url, Token: token}, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := replace(path, append(raw, '\n')); err != nil {
		return nil, err
	}
	return func() { os.Remove(path) }, nil
}

// token is what an agent presents, kept between launches.
//
// Minting a new one every time would mean the line in somebody's agent
// configuration stops working every time the application restarts, which is not
// a configuration file at all.
func token(cfg container.Config) (string, error) {
	path, err := tokenPath(cfg)
	if err != nil {
		return "", err
	}
	if kept, err := os.ReadFile(path); err == nil && len(kept) > 0 {
		return string(kept), nil
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	minted := base64.RawURLEncoding.EncodeToString(raw)

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	return minted, replace(path, []byte(minted))
}

// replace writes a file the way the rest of the application writes one: beside
// itself and renamed over the top, so a machine that dies mid-write leaves the
// previous file whole. The mode is the person's alone: it carries a secret.
func replace(path string, content []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func announcementPath(cfg container.Config) (string, error) { return beside(cfg, "agents.json") }
func tokenPath(cfg container.Config) (string, error)        { return beside(cfg, "agents.token") }

// beside is where this installation keeps its own state. A registry pointed
// somewhere chosen takes everything else with it, which is what a test and a
// second installation both need.
func beside(cfg container.Config, name string) (string, error) {
	if cfg.RegistryPath != "" {
		return filepath.Join(filepath.Dir(cfg.RegistryPath), name), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "numen", name), nil
}
