package plugincli

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestPluginValidate_InvalidID(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.json")

	badManifest := `{
  "id": "bad id without dot",
  "name": "Bad",
  "version": "1.0.0"
}`
	if err := os.WriteFile(manifestPath, []byte(badManifest), 0o644); err != nil {
		t.Fatal(err)
	}

	issues, err := Validate(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) == 0 {
		t.Fatal("expected validation issues for bad id, got none")
	}
	if !strings.Contains(issues[0], "reverse-DNS") {
		t.Errorf("unexpected issue message: %s", issues[0])
	}
}

func TestPluginValidate_InvalidPermission(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.json")

	badManifest := `{
  "id": "md.numen.test",
  "name": "Test",
  "version": "1.0.0",
  "permissions": ["unrestricted:root"]
}`
	if err := os.WriteFile(manifestPath, []byte(badManifest), 0o644); err != nil {
		t.Fatal(err)
	}

	issues, err := Validate(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) == 0 {
		t.Fatal("expected validation issue for unknown permission")
	}
	if !strings.Contains(issues[0], "unrestricted:root") {
		t.Errorf("unexpected issue message: %s", issues[0])
	}
}

func TestPluginValidate_AllowedPermissionsAndNetworkAlias(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.json")

	manifest := `{
  "id": "md.numen.valid",
  "name": "Valid",
  "version": "1.0.0",
  "permissions": ["vault:read", "vault:write", "network", "network:outbound"]
}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	issues, err := Validate(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) > 0 {
		t.Fatalf("expected no issues for allowed permissions, got: %v", issues)
	}

	// Verify rejected deprecated/unsupported permissions
	badManifest := `{
  "id": "md.numen.bad",
  "name": "Bad",
  "version": "1.0.0",
  "permissions": ["shell", "notifications", "clipboard"]
}`
	if err := os.WriteFile(manifestPath, []byte(badManifest), 0o644); err != nil {
		t.Fatal(err)
	}

	issues, err = Validate(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 3 {
		t.Fatalf("expected 3 issues for invalid permissions, got %d: %v", len(issues), issues)
	}
}

func TestPluginPackage_Success(t *testing.T) {
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "my-plugin")

	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"id": "md.numen.my-plugin", "name": "my-plugin", "version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(tmpDir, "dist", "my-plugin.zip")
	var pkgBuf bytes.Buffer
	resPath, err := Package(pluginDir, archivePath, &pkgBuf)
	if err != nil {
		t.Fatalf("unexpected package error: %v", err)
	}

	if resPath != archivePath {
		t.Errorf("got archive path %s, want %s", resPath, archivePath)
	}

	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("failed to open created zip: %v", err)
	}
	t.Cleanup(func() {
		if err := zipReader.Close(); err != nil {
			t.Error(err)
		}
	})

	var zippedFiles []string
	for _, f := range zipReader.File {
		zippedFiles = append(zippedFiles, f.Name)
	}

	if want := []string{"manifest.json"}; !slices.Equal(zippedFiles, want) {
		t.Errorf("archive holds %v, want %v", zippedFiles, want)
	}
}

// An archive holds manifest.json, bin and dist, a plugin with a frontend alone included, and no source file or older archive.
func TestPluginPackage_ContainsOnlyShippedFiles(t *testing.T) {
	pluginDir := filepath.Join(t.TempDir(), "view-sample")
	files := map[string]string{
		"manifest.json":             `{"id": "md.numen.view-sample", "name": "View", "version": "1.0.0"}`,
		"frontend/view.ts":          "source",
		"README.md":                 "readme",
		"dist/view.js":              "built",
		"dist/assets/view.css":      "built",
		"dist/old-0.9.0.zip":        "older archive",
		"bin/view-sample":           "binary",
		"node_modules/dep/index.js": "dependency",
	}
	for name, content := range files {
		path := filepath.Join(pluginDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	archivePath, err := Package(pluginDir, "", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("unexpected package error: %v", err)
	}

	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := zipReader.Close(); err != nil {
			t.Error(err)
		}
	})
	var zipped []string
	for _, f := range zipReader.File {
		zipped = append(zipped, f.Name)
	}
	slices.Sort(zipped)
	want := []string{"bin/view-sample", "dist/assets/view.css", "dist/view.js", "manifest.json"}
	if !slices.Equal(zipped, want) {
		t.Errorf("archive holds %v, want %v", zipped, want)
	}
}

func TestPluginCLI_Run(t *testing.T) {
	tmpDir := t.TempDir()
	pluginPath := filepath.Join(tmpDir, "cli-test")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Help
	code := Run([]string{"help"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected code 0 for help, got %d", code)
	}

	if err := os.MkdirAll(pluginPath, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"id": "md.numen.cli-test", "name": "CLI Test", "version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(pluginPath, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	// Validate
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"validate", pluginPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("validate failed with code %d: %s", code, stderr.String())
	}

	// Package
	stdout.Reset()
	stderr.Reset()
	archivePath := filepath.Join(tmpDir, "cli-test.zip")
	code = Run([]string{"package", pluginPath, "--out=" + archivePath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("package failed with code %d: %s", code, stderr.String())
	}
	if _, err := os.Stat(archivePath); err != nil {
		t.Errorf("archive file was not created: %v", err)
	}

	// Unknown command
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"unknown-cmd"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected code 1 for unknown command, got %d", code)
	}
}

func writeManifestFile(t *testing.T, manifest string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestPluginValidate_ReportsCoreRules(t *testing.T) {
	cases := []struct {
		name     string
		manifest string
		want     string
	}{
		{"bad semver", `{"id":"md.numen.x","name":"X","version":"1.0"}`, "semver"},
		{"leading zero version", `{"id":"md.numen.x","name":"X","version":"01.0.0"}`, "semver"},
		{"single segment id", `{"id":"foo","name":"X","version":"1.0.0"}`, "reverse-DNS"},
		{"newer protocol", `{"id":"md.numen.x","name":"X","version":"1.0.0","protocol":99}`, "protocol"},
		{"wasm backend", `{"id":"md.numen.x","name":"X","version":"1.0.0","backend":{"type":"wasm","entrypoint":"x.wasm"}}`, "must be \"process\""},
		{"negative idle", `{"id":"md.numen.x","name":"X","version":"1.0.0","lifecycle":{"idleTimeoutSeconds":-5}}`, "idleTimeoutSeconds"},
		{"setting without title", `{"id":"md.numen.x","name":"X","version":"1.0.0","settings":[{"key":"k","type":"boolean"}]}`, "title"},
		{"command without backend", `{"id":"md.numen.x","name":"X","version":"1.0.0","commands":[{"id":"run","title":"Run"}]}`, "no backend to run them"},
		{"command id in capitals", `{"id":"md.numen.x","name":"X","version":"1.0.0","backend":{"entrypoint":"bin/x"},"commands":[{"id":"Run","title":"Run"}]}`, "lowercase words joined by hyphens"},
		{"command content type that is an extension", `{"id":"md.numen.x","name":"X","version":"1.0.0","backend":{"entrypoint":"bin/x"},"commands":[{"id":"run","title":"Run","contentTypes":[".pdf"]}]}`, "invalid content type"},
		{"command argument of no type", `{"id":"md.numen.x","name":"X","version":"1.0.0","backend":{"entrypoint":"bin/x"},"commands":[{"id":"run","title":"Run","arguments":[{"name":"n","title":"N","type":"int"}]}]}`, "invalid command argument"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			issues, err := Validate(writeManifestFile(t, tc.manifest))
			if err != nil {
				t.Fatal(err)
			}
			if len(issues) == 0 || !strings.Contains(strings.Join(issues, "\n"), tc.want) {
				t.Fatalf("issues %v lack %q", issues, tc.want)
			}
		})
	}
}

func TestPluginValidate_InvalidJSON(t *testing.T) {
	issues, err := Validate(writeManifestFile(t, `{not json`))
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || !strings.Contains(issues[0], "invalid json") {
		t.Fatalf("got %v", issues)
	}
}

func TestPluginValidate_MissingEntrypointFiles(t *testing.T) {
	issues, err := Validate(writeManifestFile(t, `{"id":"md.numen.x","name":"X","version":"1.0.0",
"backend":{"type":"process","entrypoint":"bin/x"},"frontend":{"entrypoint":"dist/index.js"}}`))
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "backend entrypoint") || !strings.Contains(joined, "frontend entrypoint") {
		t.Fatalf("got %v", issues)
	}
}

func TestPluginValidate_MissingPath(t *testing.T) {
	if _, err := Validate(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatal("expected error for a missing path")
	}
}
