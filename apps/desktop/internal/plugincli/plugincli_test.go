package plugincli

import (
	"archive/zip"
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

var errWriteFailed = errors.New("write failed")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}

type countdownWriter struct {
	remaining int
	calls     int
}

func (w *countdownWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.remaining == 0 {
		return 0, errors.New("write failed")
	}
	w.remaining--
	return len(p), nil
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
  "permissions": ["vault:read", "vault:write", "network", "link:open"]
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
	if !strings.Contains(pkgBuf.String(), "Bundled 1 files into "+archivePath) {
		t.Errorf("unexpected package output: %q", pkgBuf.String())
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
		if f.Method != zip.Deflate {
			t.Errorf("file %s method = %v, want Deflate", f.Name, f.Method)
		}
		zippedFiles = append(zippedFiles, f.Name)
	}

	if want := []string{"manifest.json"}; !slices.Equal(zippedFiles, want) {
		t.Errorf("archive holds %v, want %v", zippedFiles, want)
	}

	if _, err := Package(pluginDir, archivePath, failingWriter{}); !errors.Is(err, errWriteFailed) || !strings.Contains(err.Error(), "report bundle:") {
		t.Errorf("expected wrapped report bundle error, got %v", err)
	}
	if _, err := Package(
		filepath.Join(tmpDir, "missing"),
		archivePath,
		&bytes.Buffer{},
	); !errors.Is(err, fs.ErrNotExist) ||
		strings.Contains(err.Error(), "validate plugin:") {
		t.Errorf("expected direct resolveManifestPath error without validate prefix, got %v", err)
	}
	emptyDir := t.TempDir()
	if _, err := Package(emptyDir, archivePath, &bytes.Buffer{}); !errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), "validate plugin:") {
		t.Errorf("expected validate plugin error for directory without manifest, got %v", err)
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
		"dist/assets/nested.zip":    "nested zip in subdir",
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
	want := []string{"bin/view-sample", "dist/assets/nested.zip", "dist/assets/view.css", "dist/view.js", "manifest.json"}
	if !slices.Equal(zipped, want) {
		t.Errorf("archive holds %v, want %v", zipped, want)
	}

	inBinOut := filepath.Join(pluginDir, "bin", "out.bundle")
	if _, err := Package(pluginDir, inBinOut, &bytes.Buffer{}); err != nil {
		t.Fatalf("unexpected package error for bin output: %v", err)
	}
	binReader, err := zip.OpenReader(inBinOut)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := binReader.Close(); err != nil {
			t.Error(err)
		}
	})
	for _, f := range binReader.File {
		if f.Name == "bin/out.bundle" {
			t.Fatal("archive included its own destination file")
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

	// Empty args
	stdout.Reset()
	stderr.Reset()
	if got := Run(nil, &stdout, &stderr); got != 0 || !strings.Contains(stdout.String(), "numen plugin") {
		t.Errorf("Run(nil) = %d, stdout = %q", got, stdout.String())
	}
	if got := Run(nil, failingWriter{}, &stderr); got != 1 {
		t.Errorf("Run(nil) with failing stdout = %d, want 1", got)
	}
	if got := Run([]string{"help"}, failingWriter{}, &stderr); got != 1 {
		t.Errorf("Run(help) with failing stdout = %d, want 1", got)
	}

	// Validate error cases
	stderr.Reset()
	if got := Run(
		[]string{"validate"},
		&stdout,
		&stderr,
	); got != 1 || !strings.Contains(stderr.String(), "plugin path is required") ||
		!strings.Contains(stderr.String(), "usage: numen plugin validate <path>") {
		t.Errorf("Run(validate) = %d, stderr = %q", got, stderr.String())
	}
	if got := Run([]string{"validate"}, &stdout, failingWriter{}); got != 1 {
		t.Errorf("Run(validate) with failing stderr = %d, want 1", got)
	}
	stderr.Reset()
	if got := Run(
		[]string{"validate", "--unknown-flag", pluginPath},
		&stdout,
		&stderr,
	); got != 1 || stderr.Len() == 0 ||
		strings.Contains(stderr.String(), "plugin path is required") {
		t.Errorf("Run(validate --unknown-flag) = %d, stderr = %q", got, stderr.String())
	}
	stderr.Reset()
	if got := Run(
		[]string{"validate", filepath.Join(tmpDir, "missing")},
		&stdout,
		&stderr,
	); got != 1 ||
		!strings.Contains(stderr.String(), "validation failed:") {
		t.Errorf("Run(validate missing) = %d, stderr = %q", got, stderr.String())
	}
	if got := Run([]string{"validate", pluginPath}, failingWriter{}, &stderr); got != 1 {
		t.Errorf("Run(validate) with failing stdout = %d, want 1", got)
	}
	badDir := writeManifestFile(t, `{"id":"bad","name":"B","version":"1.0.0"}`)
	stderr.Reset()
	if got := Run(
		[]string{"validate", badDir},
		&stdout,
		&stderr,
	); got != 1 || !strings.Contains(stderr.String(), "validation errors:") ||
		!strings.Contains(stderr.String(), "reverse-DNS") {
		t.Errorf("Run(validate bad) = %d, stderr = %q", got, stderr.String())
	}
	twoIssuesDir := writeManifestFile(t, `{"id":"bad","name":"B","version":"1.0"}`)
	stderr.Reset()
	if got := Run(
		[]string{"validate", twoIssuesDir},
		&stdout,
		&stderr,
	); got != 1 || !strings.Contains(stderr.String(), "reverse-DNS") ||
		!strings.Contains(stderr.String(), "semver") {
		t.Errorf("Run(validate twoIssues) = %d, stderr = %q", got, stderr.String())
	}

	// Package error cases
	stderr.Reset()
	if got := Run(
		[]string{"package"},
		&stdout,
		&stderr,
	); got != 1 || !strings.Contains(stderr.String(), "error: plugin path is required") ||
		!strings.Contains(stderr.String(), "usage: numen plugin package <path> [--out=<archive>]") {
		t.Errorf("Run(package) = %d, stderr = %q", got, stderr.String())
	}
	stderr.Reset()
	if got := Run(
		[]string{"package", "--unknown-flag", pluginPath},
		&stdout,
		&stderr,
	); got != 1 || stderr.Len() == 0 ||
		strings.Contains(stderr.String(), "plugin path is required") {
		t.Errorf("Run(package --unknown-flag) = %d, stderr = %q", got, stderr.String())
	}
	stderr.Reset()
	if got := Run([]string{"package", filepath.Join(tmpDir, "missing")}, &stdout, &stderr); got != 1 || !strings.Contains(stderr.String(), "package error:") {
		t.Errorf("Run(package missing) = %d, stderr = %q", got, stderr.String())
	}
	if got := Run([]string{"package", badDir}, &stdout, &stderr); got != 1 {
		t.Errorf("Run(package bad) = %d, want 1", got)
	}
	if got := Run([]string{"package", pluginPath, "--out=" + archivePath}, &countdownWriter{remaining: 1}, &stderr); got != 1 {
		t.Errorf("Run(package) with failing second stdout write = %d, want 1", got)
	}

	cw1 := &countdownWriter{remaining: 0}
	out1 := &output{w: cw1}
	out1.printf("first")
	out1.printf("second")
	if out1.err == nil || cw1.calls != 1 {
		t.Errorf("printf after error: err=%v calls=%d", out1.err, cw1.calls)
	}

	cw2 := &countdownWriter{remaining: 0}
	out2 := &output{w: cw2}
	out2.println("first")
	out2.println("second")
	if out2.err == nil || cw2.calls != 1 {
		t.Errorf("println after error: err=%v calls=%d", out2.err, cw2.calls)
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
		{"capability no host resolves", `{"id":"md.numen.x","name":"X","version":"1.0.0","capabilities":["agent"]}`, "invalid capability"},
		{
			"wildcard activation event",
			`{"id":"md.numen.x","name":"X","version":"1.0.0","backend":{"entrypoint":"bin/x"},"activationEvents":["*"]}`,
			"invalid activation event",
		},
		{
			"unknown activation event",
			`{"id":"md.numen.x","name":"X","version":"1.0.0","backend":{"entrypoint":"bin/x"},"activationEvents":["onCommand:run"]}`,
			"invalid activation event",
		},
		{"setting without title", `{"id":"md.numen.x","name":"X","version":"1.0.0","settings":[{"key":"k","type":"boolean"}]}`, "title"},
		{
			"command id in capitals",
			`{"id":"md.numen.x","name":"X","version":"1.0.0","backend":{"entrypoint":"bin/x"},"contributes":{"commands":[{"id":"Run","title":"Run"}]}}`,
			"each in camelCase",
		},
		{
			"command condition that names no variable",
			`{"id":"md.numen.x","name":"X","version":"1.0.0","backend":{"entrypoint":"bin/x"},"contributes":{"commands":[{"id":"note.run","title":"Run","when":"target.isRemote"}]}}`,
			`"note.run": when:`,
		},
		{
			"command argument of no type",
			`{"id":"md.numen.x","name":"X","version":"1.0.0","backend":{"entrypoint":"bin/x"},"contributes":{"commands":[{"id":"run","title":"Run","arguments":[{"name":"n","title":"N","type":"int"}]}]}}`,
			"invalid command argument",
		},
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
	dir := writeManifestFile(t, `{"id":"md.numen.x","name":"X","version":"1.0.0",
"backend": {"entrypoint": "bin/x"},"frontend": {}}`)
	issues, err := Validate(dir)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "backend entrypoint") || !strings.Contains(joined, "frontend bundle") {
		t.Fatalf("got %v", issues)
	}

	if err := os.MkdirAll(filepath.Join(dir, "backend"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "backend", "main.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "frontend"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "frontend", "index.ts"), []byte("export {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	issues, err = Validate(filepath.Join(dir, "manifest.json"))
	if err != nil || len(issues) != 0 {
		t.Fatalf("Validate with source entrypoints = %v, %v; want nil, nil", issues, err)
	}
}

func TestPluginValidate_MissingPath(t *testing.T) {
	if _, err := Validate(filepath.Join(t.TempDir(), "absent")); !errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), "stat path:") {
		t.Fatalf("expected stat path ErrNotExist for a missing path, got %v", err)
	}
	emptyDir := t.TempDir()
	if _, err := Validate(emptyDir); !errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), "read manifest at") {
		t.Fatalf("expected read manifest ErrNotExist for an empty dir without manifest.json, got %v", err)
	}
	whitespaceBackendDir := writeManifestFile(t, `{"id":"md.numen.x","name":"X","version":"1.0.0","backend":{"entrypoint":"   "}}`)
	if issues, err := Validate(whitespaceBackendDir); err != nil || len(issues) != 1 {
		t.Fatalf("expected only manifest issue for whitespace entrypoint, got %v (%v)", issues, err)
	}
}
