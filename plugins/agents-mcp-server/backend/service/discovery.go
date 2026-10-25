package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DiscoveryInfo contains connection information for external MCP clients.
type DiscoveryInfo struct {
	Name      string            `json:"name"`
	Version   string            `json:"version"`
	URL       string            `json:"url"`
	SSEURL    string            `json:"sse_url,omitempty"`
	HTTPURL   string            `json:"http_url,omitempty"`
	Token     string            `json:"token,omitempty"`
	Transport string            `json:"transport,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	PID       int               `json:"pid,omitempty"`
	UpdatedAt string            `json:"updated_at,omitempty"`
}

// writeDiscoveryFile atomically writes discovery information to the destination path.
func writeDiscoveryFile(dest string, info DiscoveryInfo) error {
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	if info.UpdatedAt == "" {
		info.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if info.PID == 0 {
		info.PID = os.Getpid()
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmpFile := dest + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
		return err
	}

	return os.Rename(tmpFile, dest)
}

// writeDiscovery writes discovery files for the server.
func writeDiscovery(targetPath string, info DiscoveryInfo) error {
	if targetPath == "" {
		return nil
	}

	if strings.HasSuffix(targetPath, ".json") {
		return writeDiscoveryFile(targetPath, info)
	}

	serviceDir := filepath.Join(targetPath, ".numen")
	agentFile := filepath.Join(serviceDir, "agent.json")
	if err := writeDiscoveryFile(agentFile, info); err != nil {
		return err
	}

	discoveryFile := filepath.Join(serviceDir, "discovery.json")
	return writeDiscoveryFile(discoveryFile, info)
}

// removeDiscovery deletes written discovery files.
func removeDiscovery(targetPath string) error {
	if targetPath == "" {
		return nil
	}

	if strings.HasSuffix(targetPath, ".json") {
		return os.Remove(targetPath)
	}

	serviceDir := filepath.Join(targetPath, ".numen")
	_ = os.Remove(filepath.Join(serviceDir, "agent.json"))
	_ = os.Remove(filepath.Join(serviceDir, "discovery.json"))
	return nil
}
