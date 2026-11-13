package codexcli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func needsShell(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fixtures are shell scripts")
	}
}

func writeScript(t *testing.T, name, body string) string {
	t.Helper()
	needsShell(t)
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func quoteShell(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
