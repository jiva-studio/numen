package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndRemoveDiscovery(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	info := DiscoveryInfo{
		Name:      "numen",
		Version:   "1.0.0",
		URL:       "http://127.0.0.1:7717/sse",
		SSEURL:    "http://127.0.0.1:7717/sse",
		HTTPURL:   "http://127.0.0.1:7717/mcp",
		Token:     "test-token",
		Transport: "sse",
		Headers: map[string]string{
			"Authorization": "Bearer test-token",
		},
	}

	if err := writeDiscovery(tmpDir, info); err != nil {
		t.Fatalf("writeDiscovery failed: %v", err)
	}

	agentPath := filepath.Join(tmpDir, ".numen", "agent.json")
	discoveryPath := filepath.Join(tmpDir, ".numen", "discovery.json")

	for _, p := range []string{agentPath, discoveryPath} {
		verifyDiscoveryFile(t, p, info)
	}

	if err := removeDiscovery(tmpDir); err != nil {
		t.Fatalf("removeDiscovery failed: %v", err)
	}

	for _, p := range []string{agentPath, discoveryPath} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("expected file %s to be removed", p)
		}
	}
}

func verifyDiscoveryFile(t *testing.T, path string, info DiscoveryInfo) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed reading %s: %v", path, err)
	}

	var readInfo DiscoveryInfo
	if err := json.Unmarshal(data, &readInfo); err != nil {
		t.Fatalf("failed unmarshaling %s: %v", path, err)
	}

	if readInfo.Name != info.Name || readInfo.Token != info.Token || readInfo.URL != info.URL {
		t.Fatalf("unexpected content in %s: %+v", path, readInfo)
	}
	if readInfo.PID == 0 {
		t.Fatalf("expected non-zero PID in %s", path)
	}
	if readInfo.UpdatedAt == "" {
		t.Fatalf("expected non-empty UpdatedAt in %s", path)
	}
}

func TestWriteDiscoverySpecificFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "custom", "custom-agent.json")

	info := DiscoveryInfo{
		Name:    "numen-custom",
		Version: "1.0.0",
		URL:     "http://127.0.0.1:9999/sse",
	}

	if err := writeDiscovery(targetFile, info); err != nil {
		t.Fatalf("writeDiscovery failed: %v", err)
	}

	data, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("failed reading target file: %v", err)
	}

	var readInfo DiscoveryInfo
	if err := json.Unmarshal(data, &readInfo); err != nil {
		t.Fatalf("failed unmarshaling: %v", err)
	}
	if readInfo.Name != "numen-custom" {
		t.Fatalf("unexpected read name: %s", readInfo.Name)
	}

	if err := removeDiscovery(targetFile); err != nil {
		t.Fatalf("removeDiscovery failed: %v", err)
	}
	if _, err := os.Stat(targetFile); !os.IsNotExist(err) {
		t.Fatal("expected file to be deleted")
	}
}

func TestRemoveDiscoveryNonExistent(t *testing.T) {
	t.Parallel()

	if err := removeDiscovery(""); err != nil {
		t.Fatalf("removeDiscovery empty error: %v", err)
	}

	if err := removeDiscovery(filepath.Join(t.TempDir(), "does-not-exist")); err != nil {
		t.Fatalf("removeDiscovery non-existent error: %v", err)
	}
}
