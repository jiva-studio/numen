package container

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

const (
	rowsPlugin     = "md.numen.sample.rows"
	rowsConnection = "c1f0a9e20"
	otherRowsID    = "c77b01d40"
)

const rowsManifest = `{
	"id": "` + rowsPlugin + `", "name": "Rows", "version": "1.0.0", "protocol": 1,
	"capabilities": [], "targets": ["desktop.editor"],
	"settings": [{"key": "connections", "type": "connections", "title": "Connections", "operation": "complete", "max": 3}]
}`

const rowsInstallation = `{
	"connections": [
		{"id": "c1f0a9e20", "type": "gone.plugin/x", "name": "Work", "revision": 1},
		{"id": "c77b01d40", "type": "gone.plugin/x", "name": "Ollama", "revision": 1}
	],
	"` + rowsPlugin + `": {"connections": [
		{"connection": "c1f0a9e20", "model": "a"},
		{"connection": "c77b01d40", "model": "b"},
		{"connection": "c1f0a9e20", "model": "c"}
	]}
}`

type noNotices struct{}

func (noNotices) ReportSetAside(string, string, connection.FailureReason) {}

func openRowsInstallation(t *testing.T) (Config, string) {
	t.Helper()
	plugins := t.TempDir()
	dir := filepath.Join(plugins, "rows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(rowsManifest), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(path, []byte(rowsInstallation), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := NewConfig(Config{Paths: Paths{SettingsPath: path}, PluginHost: newPluginHost(t, "desktop.editor", plugins)})
	stored, err := cfg.Settings()
	if err != nil {
		t.Fatal(err)
	}
	cfg = cfg.SetSettings(stored)
	t.Cleanup(func() { UnsubscribeHost(cfg) })
	return cfg, path
}

func TestTheConnectionsListWhereTheirRowsAre(t *testing.T) {
	cfg, _ := openRowsInstallation(t)

	entries, err := openConnectionUseCases(plugins(cfg)).List.Execute(t.Context())

	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || len(entries[0].Uses) != 1 || entries[0].Uses[0].PluginID != rowsPlugin || entries[0].Uses[0].PluginName != "Rows" || entries[0].Uses[0].Operation != "complete" {
		t.Errorf("uses of the first connection = %+v", entries[0].Uses)
	}
}

func TestDeletingAConnectionRemovesTheRowsThatNameItFromTheFile(t *testing.T) {
	cfg, path := openRowsInstallation(t)

	removed, err := openConnectionUseCases(plugins(cfg)).Delete.Execute(t.Context(), rowsConnection)

	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || removed[0].PluginID != rowsPlugin {
		t.Errorf("removed uses = %+v", removed)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), rowsConnection) {
		t.Errorf("the file still names the deleted connection:\n%s", raw)
	}
	if !strings.Contains(string(raw), otherRowsID) {
		t.Errorf("the row of the other connection is gone:\n%s", raw)
	}
}

func TestAWriteOfRowsThatNameNoConnectionIsRefused(t *testing.T) {
	cfg, _ := openRowsInstallation(t)

	err := GetSettingsWriter(cfg)([]settingsdomain.Setting{{Path: []string{rowsPlugin, "connections"}, JSON: `[{"connection":"c00000000","model":"m"}]`}})

	if err == nil {
		t.Error("rows that name a connection that does not exist were written")
	}
}

func TestAPluginAsksTheRowsOfItsOwnSettingForAConnection(t *testing.T) {
	cfg, _ := openRowsInstallation(t)
	setModelCalls(plugins(cfg), noNotices{})

	_, err := cfg.PluginHost.Endpoint.GetModelCalls().Complete(t.Context(), rowsPlugin, modelcall.CompleteRequest{}, modelcall.PriorityBackground)

	var failed *connection.AllFailedError
	if !errors.As(err, &failed) {
		t.Fatalf("err = %v, want every row set aside: the rows are read from the settings", err)
	}
	tried := make([]string, 0, len(failed.Failures))
	for _, one := range failed.Failures {
		tried = append(tried, one.ConnectionName+": "+one.Reason)
	}
	want := []string{rowsConnection + ": type unavailable", otherRowsID + ": type unavailable", rowsConnection + ": type unavailable"}
	if !slices.Equal(tried, want) {
		t.Errorf("rows tried = %v, want %v in the order of the rows", tried, want)
	}
}

func TestAPluginWithoutRowsGetsNoConnection(t *testing.T) {
	cfg, _ := openRowsInstallation(t)
	setModelCalls(plugins(cfg), noNotices{})

	_, err := cfg.PluginHost.Endpoint.GetModelCalls().Complete(t.Context(), "md.numen.sample.other", modelcall.CompleteRequest{}, modelcall.PriorityBackground)

	if !errors.Is(err, connection.ErrNoConnection) {
		t.Errorf("err = %v, want ErrNoConnection", err)
	}
}
