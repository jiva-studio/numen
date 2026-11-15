package manifestjson_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func TestListPackageIssuesAcceptsTheBuiltFilesOrTheirSources(t *testing.T) {
	m, err := manifestjson.Decode([]byte(`{"id":"org.example.tides","name":"Tides","version":"1.0.0",
"backend": {"entrypoint": "bin/tides"},"frontend": {}}`))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	joined := strings.Join(manifestjson.ListPackageIssues(m, dir), "\n")
	if !strings.Contains(joined, "backend entrypoint") || !strings.Contains(joined, "frontend bundle") {
		t.Fatalf("issues of an empty folder: %s", joined)
	}

	for path, text := range map[string]string{"backend/main.go": "package main", "frontend/index.ts": "export {}"} {
		if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, path), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if issues := manifestjson.ListPackageIssues(m, dir); len(issues) != 0 {
		t.Fatalf("issues of a folder with the sources: %v", issues)
	}
}

func TestListPackageIssuesAsksNothingOfABlankEntrypoint(t *testing.T) {
	m, err := manifestjson.Decode([]byte(`{"id":"org.example.tides","name":"Tides","version":"1.0.0","backend":{"entrypoint":"   "}}`))
	if err != nil {
		t.Fatal(err)
	}
	if issues := manifestjson.ListPackageIssues(m, t.TempDir()); len(issues) != 0 {
		t.Fatalf("issues = %v, want none", issues)
	}

	themed, err := manifestjson.Decode([]byte(`{"id":"org.example.tides","name":"Tides","version":"1.0.0",
		"contributes":{"themes":[{"id":"day","file":"day.css"},{"id":"night","file":"night.css"}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	themeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(themeDir, "day.css"), []byte(":root{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	themeIssues := manifestjson.ListPackageIssues(themed, themeDir)
	if len(themeIssues) != 1 || !strings.Contains(themeIssues[0], `theme "night"`) {
		t.Fatalf("theme issues = %v, want missing night theme", themeIssues)
	}
}
