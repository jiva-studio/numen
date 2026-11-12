// Package toolconfig writes the file that names the vault's tool server for a run of the program.
package toolconfig

import (
	"encoding/json"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
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
	// gomutants:disable-next-line BRANCH_IF reason="Marshaling a static struct of string fields and string maps cannot fail."
	if err != nil {
		return "", fmt.Errorf("marshal mcp config: %w", err)
	}

	return agentrun.WriteTempFile("numen-tools-*.json", data)
}

// Remove deletes the file; a path of none or of a file that is gone is not an error.
func (Files) Remove(path string) error {
	return agentrun.RemoveFile(path)
}
