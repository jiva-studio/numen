// Package toolconfig writes the file that names the vault's tool server for a run of the program.
package toolconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// Files keeps the tool configuration files of runs in files only this user can read, because the
// token grants access to the whole vault and every user of the machine can read a command line.
type Files struct{}

// Write stores the configuration and returns the path of the file, or none when the address or the
// token is empty.
func (Files) Write(url, token string) (string, error) {
	if url == "" || token == "" {
		return "", nil
	}
	type server struct {
		Type    string            `json:"type"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}
	config := struct {
		Servers map[string]server `json:"mcpServers"`
	}{
		Servers: map[string]server{
			"numen": {
				Type:    "http",
				URL:     url,
				Headers: map[string]string{"Authorization": "Bearer " + token},
			},
		},
	}
	data, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshal mcp config: %w", err)
	}
	return writeTempFile("numen-tools-*.json", data)
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
