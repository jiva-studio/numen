package container_test

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
)

const (
	eventPlugin = "md.numen.eventful"
	// wait is how long a test gives an event that has to arrive.
	wait = 5 * time.Second
	// quiet is how long a test listens for an event that must not arrive.
	quiet = 600 * time.Millisecond
)

// recordingHost is a plugin host that records what the container tells it, and passes the rest to
// the real host.
type recordingHost struct {
	pluginport.SettingsPush
	apply    settingsport.PluginSettingApplier
	mu       sync.Mutex
	updates  []string
	profiles []string
}

func (r *recordingHost) applySetting(ctx context.Context, pluginID, key string, value any) error {
	r.mu.Lock()
	r.updates = append(r.updates, pluginID+"."+key)
	r.mu.Unlock()
	return r.apply(ctx, pluginID, key, value)
}

func (r *recordingHost) SetPerformanceProfile(ctx context.Context, profile string) {
	r.mu.Lock()
	r.profiles = append(r.profiles, profile)
	r.mu.Unlock()
	r.SettingsPush.SetPerformanceProfile(ctx, profile)
}

func (r *recordingHost) listUpdates() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.updates)
}

func (r *recordingHost) listProfiles() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.profiles)
}

func writeEventfulPlugin(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "eventful")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"id": "` + eventPlugin + `",
		"name": "Eventful",
		"version": "1.0.0",
		"protocol": 1,
		"capabilities": [],
		"targets": ["desktop.editor"],
		"settings": [{"key": "mode", "type": "string", "title": "Mode", "default": "a"}]
	}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func openInstallation(t *testing.T, body string) (container.Config, string, *recordingHost) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	lent := newExternalPluginHost(t, "desktop.editor", writeEventfulPlugin(t))
	host := &recordingHost{apply: lent.ApplySetting, SettingsPush: lent.Configuration.SettingsPush}
	lent.ApplySetting, lent.Configuration.SettingsPush = host.applySetting, host
	cfg := container.NewConfig(container.Config{
		PluginHost: lent,
		Paths:      container.Paths{SettingsPath: path},
	})
	cfg = cfg.SetSettings(container.DefaultSettings())
	t.Cleanup(func() { container.UnsubscribeHost(cfg) })
	return cfg, path, host
}

// listener is one reader of the setting-changed event, such as a window or a plugin.
type listener struct {
	changes <-chan []settings.Setting
}

func listen(t *testing.T, cfg container.Config) listener {
	t.Helper()
	changes, stop := cfg.SubscribeSettingChanges()
	t.Cleanup(stop)
	return listener{changes: changes}
}

func (l listener) await(t *testing.T) []settings.Setting {
	t.Helper()
	select {
	case batch, open := <-l.changes:
		if !open {
			t.Fatal("the subscription closed before a change arrived")
		}
		return batch
	case <-time.After(wait):
		t.Fatal("no change arrived")
		return nil
	}
}

func (l listener) expectNothing(t *testing.T) {
	t.Helper()
	select {
	case batch, open := <-l.changes:
		if open {
			t.Fatalf("a change arrived that should not have: %v", batch)
		}
	case <-time.After(quiet):
	}
}

func expectChange(t *testing.T, batch []settings.Setting, path []string, json string) {
	t.Helper()
	if len(batch) != 1 || !slices.Equal(batch[0].Path, path) || batch[0].JSON != json {
		t.Errorf("the change is %v, want %v set to %s", batch, path, json)
	}
}

func TestAWriteByTheSettingsPageReachesEveryReaderOnceWithItsDeclaredPath(t *testing.T) {
	cfg, _, _ := openInstallation(t, `{}`)
	window, other := listen(t, cfg), listen(t, cfg)

	err := container.GetSettingsWriter(cfg)([]settings.Setting{
		{Path: []string{"app", "appearance", "mode"}, JSON: `"dark"`},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, reader := range []listener{window, other} {
		expectChange(t, reader.await(t), []string{"app", "appearance", "mode"}, `"dark"`)
		reader.expectNothing(t)
	}
}

func TestAWriteOfSeveralSettingsArrivesAsOneBatch(t *testing.T) {
	cfg, _, _ := openInstallation(t, `{}`)
	reader := listen(t, cfg)

	err := container.GetSettingsWriter(cfg)([]settings.Setting{
		{Path: []string{"app", "appearance", "mode"}, JSON: `"dark"`},
		{Path: []string{"app", "appearance", "textScale"}, JSON: `1.2`},
	})
	if err != nil {
		t.Fatal(err)
	}

	batch := reader.await(t)
	if len(batch) != 2 {
		t.Fatalf("the batch holds %v", batch)
	}
	reader.expectNothing(t)
}

func TestAWriteByAPluginReachesEveryReaderOnceWithItsDeclaredPath(t *testing.T) {
	cfg, _, _ := openInstallation(t, `{}`)
	window, other := listen(t, cfg), listen(t, cfg)

	err := container.GetSettingsWriter(cfg)([]settings.Setting{
		{Path: []string{eventPlugin, "mode"}, JSON: `"b"`},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, reader := range []listener{window, other} {
		expectChange(t, reader.await(t), []string{eventPlugin, "mode"}, `"b"`)
		reader.expectNothing(t)
	}
}

func TestAWriteTheCheckRejectsReachesNoReader(t *testing.T) {
	cfg, _, _ := openInstallation(t, `{}`)
	reader := listen(t, cfg)

	if err := container.GetSettingsWriter(cfg)([]settings.Setting{
		{Path: []string{eventPlugin, "undeclared"}, JSON: `"x"`},
	}); err == nil {
		t.Fatal("a key nobody declares was written")
	}

	reader.expectNothing(t)
}

func TestAWriteThatChangesNoValueReachesNoReader(t *testing.T) {
	cfg, _, _ := openInstallation(t, `{"app":{"appearance":{"mode":"dark"}}}`)
	reader := listen(t, cfg)

	err := container.GetSettingsWriter(cfg)([]settings.Setting{
		{Path: []string{"app", "appearance", "mode"}, JSON: `"dark"`},
	})
	if err != nil {
		t.Fatal(err)
	}

	reader.expectNothing(t)
}

func TestAHandEditOfTheFileReachesEveryReaderOnceWithItsDeclaredPath(t *testing.T) {
	cfg, path, _ := openInstallation(t, `{"app":{"appearance":{"mode":"light"}}}`)
	window, other := listen(t, cfg), listen(t, cfg)

	if err := os.WriteFile(path, []byte(`{"app":{"appearance":{"mode":"dark"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, reader := range []listener{window, other} {
		expectChange(t, reader.await(t), []string{"app", "appearance", "mode"}, `"dark"`)
		reader.expectNothing(t)
	}
}

func TestAHandEditOfAPluginSettingReachesEveryReaderOnce(t *testing.T) {
	cfg, path, _ := openInstallation(t, `{}`)
	reader := listen(t, cfg)

	if err := os.WriteFile(path, []byte(`{"`+eventPlugin+`":{"mode":"c"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	expectChange(t, reader.await(t), []string{eventPlugin, "mode"}, `"c"`)
	reader.expectNothing(t)
}

func TestAHandEditThatKeepsEveryValueReachesNoReader(t *testing.T) {
	cfg, path, _ := openInstallation(t, `{"app":{"appearance":{"mode":"dark"}}}`)
	reader := listen(t, cfg)

	if err := os.WriteFile(path, []byte("{\n  \"app\": {\"appearance\": {\"mode\": \"dark\"}}\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	reader.expectNothing(t)
}

func TestAHandEditOfAKeyNoOwnerDeclaresReachesNoReader(t *testing.T) {
	cfg, path, _ := openInstallation(t, `{}`)
	reader := listen(t, cfg)

	body := `{"app":{"appearance":{"mistyped":"x"}},"` + eventPlugin + `":{"nothing":1},"md.numen.nobody":{"mode":"x"}}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	reader.expectNothing(t)
}

func TestAHandEditToAValueOutOfBoundsReachesNoReader(t *testing.T) {
	cfg, path, _ := openInstallation(t, `{}`)
	reader := listen(t, cfg)

	if err := os.WriteFile(path, []byte(`{"app":{"appearance":{"interfaceScale":9}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	reader.expectNothing(t)
}

func TestAHandEditWithBrokenJSONReachesNoReaderAndTheNextEditDoes(t *testing.T) {
	cfg, path, _ := openInstallation(t, `{"app":{"appearance":{"mode":"light"}}}`)
	reader := listen(t, cfg)

	if err := os.WriteFile(path, []byte(`{"app":{"appearance":`), 0o600); err != nil {
		t.Fatal(err)
	}
	reader.expectNothing(t)

	if err := os.WriteFile(path, []byte(`{"app":{"appearance":{"mode":"dark"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	expectChange(t, reader.await(t), []string{"app", "appearance", "mode"}, `"dark"`)
}

func TestAHandEditReachesTheRunningPluginThroughTheSameEvent(t *testing.T) {
	cfg, path, host := openInstallation(t, `{}`)
	reader := listen(t, cfg)

	if err := os.WriteFile(path, []byte(`{"`+eventPlugin+`":{"mode":"c"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	reader.await(t)

	waitFor(t, func() bool { return len(host.listUpdates()) > 0 })
	if got := host.listUpdates(); !slices.Equal(got, []string{eventPlugin + ".mode"}) {
		t.Errorf("the plugin host received %v", got)
	}
}

func TestAHandEditOfThePerformanceProfileReachesThePluginHostOnce(t *testing.T) {
	cfg, path, host := openInstallation(t, `{}`)
	reader := listen(t, cfg)
	before := len(host.listProfiles())

	if err := os.WriteFile(path, []byte(`{"app":{"performance":{"profile":"eco"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	reader.await(t)

	waitFor(t, func() bool { return len(host.listProfiles()) > before })
	time.Sleep(quiet)
	if got := host.listProfiles()[before:]; !slices.Equal(got, []string{"eco"}) {
		t.Errorf("the plugin host was given the profiles %v", got)
	}
}

func TestAWriteByAPluginReachesThePluginHostOnce(t *testing.T) {
	cfg, _, host := openInstallation(t, `{}`)
	reader := listen(t, cfg)

	if err := container.GetSettingsWriter(cfg)([]settings.Setting{
		{Path: []string{eventPlugin, "mode"}, JSON: `"b"`},
	}); err != nil {
		t.Fatal(err)
	}
	reader.await(t)

	waitFor(t, func() bool { return len(host.listUpdates()) > 0 })
	time.Sleep(quiet)
	if got := host.listUpdates(); !slices.Equal(got, []string{eventPlugin + ".mode"}) {
		t.Errorf("the plugin host received %v", got)
	}
}

func TestAReaderThatStoppedListeningReceivesNothingMore(t *testing.T) {
	cfg, _, _ := openInstallation(t, `{}`)
	changes, stop := cfg.SubscribeSettingChanges()
	stop()

	err := container.GetSettingsWriter(cfg)([]settings.Setting{
		{Path: []string{"app", "appearance", "mode"}, JSON: `"dark"`},
	})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case batch, open := <-changes:
		if open {
			t.Errorf("a change arrived after the reader stopped: %v", batch)
		}
	case <-time.After(quiet):
		t.Error("the channel of a reader that stopped listening was not closed")
	}
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(wait)
	for !condition() {
		select {
		case <-ticker.C:
		case <-timeout:
			t.Fatal("the condition was never met")
		}
	}
}

func TestNoSettingHasAPathOfItsOwnInTheContainer(t *testing.T) {
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
			if declared, ok := node.(*ast.FuncDecl); ok && declared.Name.Name == "notifyPerformanceProfile" {
				t.Errorf("%s still has %s, a path for one setting beside the event", file, declared.Name.Name)
			}
			return true
		})
	}
}

func TestAWriteOfThePerformanceProfileReachesThePluginHostOnce(t *testing.T) {
	cfg, _, host := openInstallation(t, `{}`)
	reader := listen(t, cfg)
	before := len(host.listProfiles())

	if err := container.GetSettingsWriter(cfg)([]settings.Setting{
		{Path: []string{"app", "performance", "profile"}, JSON: `"eco"`},
	}); err != nil {
		t.Fatal(err)
	}
	reader.await(t)

	waitFor(t, func() bool { return len(host.listProfiles()) > before })
	time.Sleep(quiet)
	if got := host.listProfiles()[before:]; !slices.Equal(got, []string{"eco"}) {
		t.Errorf("the plugin host was given the profiles %v", got)
	}
}
