package container_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
)

const importerID = "md.numen.tools.media.importer"

// importerManifest declares two text settings, a number, and a
// switch, the way the importer plugin declares its settings.
const importerManifest = `{
	"id": "md.numen.tools.media.importer", "name": "Importer", "version": "1.0.0",
	"capabilities": [], "targets": ["desktop.editor"],
	"services": ["numen.v1.AgentService"], "backend": {"entrypoint": "bin/x"},
	"settings": [
		{"key": "ytDlp", "type": "string", "title": "yt-dlp", "default": ""},
		{"key": "ffmpeg", "type": "string", "title": "ffmpeg", "default": ""},
		{"key": "retries", "type": "number", "title": "Retries", "default": 3},
		{"key": "verbose", "type": "boolean", "title": "Verbose", "default": false}
	]
}`

func installationWith(t *testing.T, content string) (container.Config, string) {
	t.Helper()
	plugins := t.TempDir()
	dir := filepath.Join(plugins, importerID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(importerManifest), 0o600); err != nil {
		t.Fatal(err)
	}
	host := pluginhost.New(testsupport.GetMachine(), "desktop.editor", plugins)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return container.NewConfig(container.Config{
		PluginHost: container.AdaptPluginHost((*container.PluginRoles)(host)),
		Paths:      container.Paths{SettingsPath: path},
	}), path
}

func countNotices(notices []string, words ...string) int {
	count := 0
	for _, notice := range notices {
		found := true
		for _, word := range words {
			found = found && strings.Contains(notice, word)
		}
		if found {
			count++
		}
	}
	return count
}

func TestAValueOfTheWrongTypeInTheShellIsDroppedAndTheFileReads(t *testing.T) {
	cfg, _ := installationWith(t, `{"app":{"appearance":{"textScale":"big"}}}`)
	got, err := cfg.Settings()
	if err != nil {
		t.Fatalf("the read stops: %v", err)
	}
	if value, _ := got.GetValue("app", "appearance", "textScale"); value != 1.0 {
		t.Errorf("the text scale in force is %v, want the default", value)
	}
	if len(got.Notices) != 1 || countNotices(got.Notices, "appearance.textScale") != 1 {
		t.Errorf("the notices are %q", got.Notices)
	}
}

func TestAnObjectWhereAPluginDeclaresTextIsDroppedAndTheFileReads(t *testing.T) {
	cfg, _ := installationWith(t, `{"`+importerID+`":{"ytDlp":{"environment":{"A":"b"}},"retries":5}}`)
	got, err := cfg.Settings()
	if err != nil {
		t.Fatalf("the read stops: %v", err)
	}
	owner := got.GetOwner(importerID)
	if _, found := owner["ytDlp"]; found || owner["retries"] != 5.0 {
		t.Errorf("the object of the plugin is %v", owner)
	}
	if len(got.Notices) != 1 || countNotices(got.Notices, importerID+".ytDlp", "wrong type") != 1 {
		t.Errorf("the notices are %q", got.Notices)
	}
}

func TestATextWhereAPluginDeclaresANumberIsDroppedAndTheFileReads(t *testing.T) {
	cfg, _ := installationWith(t, `{"`+importerID+`":{"retries":"many","verbose":true}}`)
	got, err := cfg.Settings()
	if err != nil {
		t.Fatalf("the read stops: %v", err)
	}
	owner := got.GetOwner(importerID)
	if _, found := owner["retries"]; found || owner["verbose"] != true {
		t.Errorf("the object of the plugin is %v", owner)
	}
	if len(got.Notices) != 1 || countNotices(got.Notices, importerID+".retries") != 1 {
		t.Errorf("the notices are %q", got.Notices)
	}
}

func TestATextWhereThePluginIsSwitchedOnOrOffIsDroppedAndTheFileReads(t *testing.T) {
	cfg, _ := installationWith(t, `{"`+importerID+`":{"enabled":"yes","retries":2}}`)
	got, err := cfg.Settings()
	if err != nil {
		t.Fatalf("the read stops: %v", err)
	}
	owner := got.GetOwner(importerID)
	if _, found := owner["enabled"]; found || owner["retries"] != 2.0 {
		t.Errorf("the object of the plugin is %v", owner)
	}
	if len(got.Notices) != 1 || countNotices(got.Notices, importerID+".enabled") != 1 {
		t.Errorf("the notices are %q", got.Notices)
	}
}

func TestTheFileIsUnchangedByReadingValuesOfTheWrongType(t *testing.T) {
	const body = `{"app":{"appearance":{"textScale":"big"}},"` + importerID +
		`":{"ytDlp":{"a":1},"retries":"many","enabled":"yes"}}`
	cfg, path := installationWith(t, body)
	if _, err := cfg.Settings(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != body {
		t.Errorf("the file is now:\n%s", raw)
	}
}

func TestAFileThatIsNoJSONStillStopsTheReadAndNamesTheFile(t *testing.T) {
	cfg, path := installationWith(t, `{"app": {"appearance": `)
	_, err := cfg.Settings()
	if err == nil {
		t.Fatal("the file was read")
	}
	if !strings.Contains(err.Error(), filepath.Base(path)) || !strings.Contains(err.Error(), "byte") {
		t.Errorf("the error names no file and no position: %v", err)
	}
}

func TestOldKeysOfTheFileAreIgnoredWithNoticesThatNameThemAndNothingIsCarriedIntoAPlugin(t *testing.T) {
	const old = `{"importing":{"yt_dlp":{"environment":{"A":"b"}},"ffmpeg":{"command":["ffmpeg","-y"]}},"review":{"day_start_time":"05:00"}}`
	cfg, path := installationWith(t, old)
	got, err := cfg.Settings()
	if err != nil {
		t.Fatalf("the read stops: %v", err)
	}
	if owner := got.GetOwner(importerID); len(owner) != 0 {
		t.Errorf("the object of the plugin holds %v", owner)
	}
	for _, key := range []string{"importing", "review"} {
		if countNotices(got.Notices, key, "ignored") != 1 {
			t.Errorf("no notice names %s: %q", key, got.Notices)
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != old {
		t.Errorf("the file is now:\n%s", raw)
	}
}

func TestAnObjectInsideTheKeyPluginsIsIgnoredWithANotice(t *testing.T) {
	cfg, _ := installationWith(t, `{"plugins":{"`+importerID+`":{"ytDlp":{"environment":{"A":"b"}}}}}`)
	got, err := cfg.Settings()
	if err != nil {
		t.Fatalf("the read stops: %v", err)
	}
	if value, found := got.GetValue(importerID, "ytDlp"); found {
		t.Errorf("the object is in force as the yt-dlp program: %v", value)
	}
	if countNotices(got.Notices, "plugins", "ignored") != 1 {
		t.Errorf("the notices are %q", got.Notices)
	}
}

func TestNoPathCarriesAnOldKeyIntoASettingOfAPlugin(t *testing.T) {
	removed := []string{"MigratePluginSettings", "collectPluginSettingsMigrations", "migrateOneSetting"}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			if declared, ok := node.(*ast.FuncDecl); ok && slices.Contains(removed, declared.Name.Name) {
				t.Errorf("%s still has %s", file, declared.Name.Name)
			}
			return true
		})
	}
}
