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

func TestPluginCreate_Native(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "sample-native")

	opts := CreateOptions{
		Name:      "sample-native",
		Type:      "native",
		Template:  "",
		OutputDir: outDir,
	}

	var buf bytes.Buffer
	if err := Create(opts, &buf); err != nil {
		t.Fatalf("unexpected error creating native plugin: %v", err)
	}

	for _, file := range []string{
		"manifest.json",
		"go.mod",
		"backend/main.go",
		"Makefile",
		"README.md",
	} {
		path := filepath.Join(outDir, file)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s to exist: %v", file, err)
		}
	}

	issues, err := Validate(outDir)
	if err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}
	if len(issues) > 0 {
		t.Fatalf("validation issues found on created plugin: %v", issues)
	}
}

func TestPluginCreate_Frontend(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "sample-frontend")

	opts := CreateOptions{
		Name:      "sample-frontend",
		Type:      "frontend",
		Template:  "",
		OutputDir: outDir,
	}

	var buf bytes.Buffer
	if err := Create(opts, &buf); err != nil {
		t.Fatalf("unexpected error creating frontend plugin: %v", err)
	}

	for _, file := range []string{
		"manifest.json",
		"frontend/index.ts",
		"frontend/Viewer.vue",
		"frontend/package.json",
		"frontend/vite.config.ts",
		"Makefile",
	} {
		path := filepath.Join(outDir, file)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s to exist: %v", file, err)
		}
	}

	issues, err := Validate(outDir)
	if err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}
	if len(issues) > 0 {
		t.Fatalf("validation issues found: %v", issues)
	}
}

func TestPluginCreate_ReaderTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "sample-reader")

	opts := CreateOptions{
		Name:      "sample-reader",
		Type:      "native",
		Template:  "reader",
		OutputDir: outDir,
	}

	var buf bytes.Buffer
	if err := Create(opts, &buf); err != nil {
		t.Fatalf("unexpected error creating reader plugin: %v", err)
	}

	manifestBytes, err := os.ReadFile(filepath.Join(outDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifestStr := string(manifestBytes)

	if !strings.Contains(manifestStr, `"document"`) {
		t.Errorf("expected document capability in reader manifest: %s", manifestStr)
	}
	if !strings.Contains(manifestStr, `"onFile:*.sample-reader"`) {
		t.Errorf("expected onFile activation in reader manifest: %s", manifestStr)
	}

	issues, err := Validate(outDir)
	if err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}
	if len(issues) > 0 {
		t.Fatalf("validation issues found: %v", issues)
	}
}

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
	if !strings.Contains(issues[0], "plugin id") {
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

	opts := CreateOptions{
		Name:      "my-plugin",
		Type:      "native",
		Template:  "",
		OutputDir: pluginDir,
	}

	var createBuf bytes.Buffer
	if err := Create(opts, &createBuf); err != nil {
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
	defer zipReader.Close()

	var zippedFiles []string
	for _, f := range zipReader.File {
		zippedFiles = append(zippedFiles, f.Name)
	}

	for _, expected := range []string{"manifest.json", "go.mod", "backend/main.go", "README.md"} {
		if !slices.Contains(zippedFiles, expected) {
			t.Errorf("expected archive to contain %s, got: %v", expected, zippedFiles)
		}
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

	// Create
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"create", "cli-test", "--type=native", "--out=" + pluginPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("create failed with code %d: %s", code, stderr.String())
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

func TestPluginCreate_ReaderCompiles(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "sample-reader")

	opts := CreateOptions{
		Name:      "sample-reader",
		Type:      "native",
		Template:  "reader",
		OutputDir: outDir,
	}

	var buf bytes.Buffer
	if err := Create(opts, &buf); err != nil {
		t.Fatalf("unexpected error creating reader plugin: %v", err)
	}

	goModBytes, err := os.ReadFile(filepath.Join(outDir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goModBytes), "replace github.com/jiva-studio/numen/modules/libs/protocol") {
		t.Errorf("expected protocol replace directive in go.mod")
	}
}

func TestValidateExistingRepoPlugins(t *testing.T) {
	pluginsDir := "../../../../plugins"
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		t.Skip("plugins directory not found relative to test")
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pluginPath := filepath.Join(pluginsDir, entry.Name())
		if _, err := os.Stat(filepath.Join(pluginPath, "manifest.json")); err != nil {
			continue
		}

		issues, err := Validate(pluginPath)
		if err != nil {
			t.Errorf("error validating %s: %v", entry.Name(), err)
			continue
		}
		if len(issues) > 0 {
			t.Errorf("validation issues in existing plugin %s: %v", entry.Name(), issues)
		}
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
		{"newer protocol", `{"id":"md.numen.x","name":"X","version":"1.0.0","protocol":99}`, "protocol"},
		{"wasm backend", `{"id":"md.numen.x","name":"X","version":"1.0.0","backend":{"type":"wasm","entrypoint":"x.wasm"}}`, "backend"},
		{"negative idle", `{"id":"md.numen.x","name":"X","version":"1.0.0","lifecycle":{"idleTimeoutSeconds":-5}}`, "idleTimeoutSeconds"},
		{"setting without title", `{"id":"md.numen.x","name":"X","version":"1.0.0","settings":[{"key":"k","type":"boolean"}]}`, "title"},
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

func TestPluginCreate_ManifestCarriesProtocolAndNoMode(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "p")
	if err := Create(CreateOptions{Name: "p", Type: "frontend", OutputDir: outDir}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(outDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"protocol": 1`) {
		t.Errorf("manifest lacks protocol: %s", raw)
	}
	if strings.Contains(string(raw), `"mode"`) || strings.Contains(string(raw), `"backend"`) {
		t.Errorf("frontend manifest carries mode or backend: %s", raw)
	}
}
