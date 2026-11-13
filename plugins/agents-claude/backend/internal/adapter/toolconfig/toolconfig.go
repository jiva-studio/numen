// Package toolconfig writes the file that names the vault's tool server and the person's own MCP servers
// for a run of the program.
package toolconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
)

// serverName is the name the vault's tool server has in the file; a server of the person by that name
// is left out.
const serverName = "numen"

// Files keeps the tool configuration files of runs in files only this user can read, because the
// token grants access to the whole vault and every user of the machine can read a command line.
type Files struct{}

// Write stores the configuration and returns the path of the file and the names of the person's own
// servers in it, or no path when the address or the token is empty. The person's servers are the user
// servers of Claude Code, in .claude.json of CLAUDE_CONFIG_DIR, else of the home folder; a file that
// cannot be read names none.
func (Files) Write(url, token string) (string, []string, error) {
	if url == "" || token == "" {
		return "", nil, nil
	}
	servers := readUserServers()
	vault, err := json.Marshal(map[string]any{
		"type": "http", "url": url, "headers": map[string]string{"Authorization": "Bearer " + token},
	})
	if err != nil {
		return "", nil, fmt.Errorf("marshal mcp config: %w", err)
	}
	names := slices.Sorted(maps.Keys(servers))
	servers[serverName] = vault
	data, err := json.Marshal(map[string]any{"mcpServers": servers})
	if err != nil {
		return "", nil, fmt.Errorf("marshal mcp config: %w", err)
	}
	path, err := writeTempFile("numen-tools-*.json", data)
	if err != nil {
		return "", nil, err
	}
	return path, names, nil
}

func readUserServers() map[string]json.RawMessage {
	servers := map[string]json.RawMessage{}
	dir := os.Getenv("CLAUDE_CONFIG_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return servers
		}
		dir = home
	}
	raw, err := os.ReadFile(filepath.Join(dir, ".claude.json"))
	if err != nil {
		return servers
	}
	var config struct {
		Servers map[string]json.RawMessage `json:"mcpServers"`
	}
	if json.Unmarshal(raw, &config) != nil {
		return servers
	}
	for name, server := range config.Servers {
		if name != serverName {
			servers[name] = server
		}
	}
	return servers
}

// Remove deletes the file; a path of none or of a file that is gone is not an error.
func (Files) Remove(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return nil
}

func writeTempFile(pattern string, data []byte) (path string, err error) {
	file, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("create temp config: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close config: %w", closeErr))
		}
		if err != nil {
			if removeErr := os.Remove(file.Name()); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
				err = errors.Join(err, fmt.Errorf("remove config: %w", removeErr))
			}
			path = ""
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		return "", fmt.Errorf("chmod config: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}
	return file.Name(), nil
}
