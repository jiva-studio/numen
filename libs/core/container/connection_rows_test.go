package container_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/settingsport"
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

type callsHost struct {
	pluginport.PluginManager
	calls connectionport.ModelCalls
}

func (h *callsHost) SetModelCalls(calls connectionport.ModelCalls) { h.calls = calls }

type noNotices struct{}

func (noNotices) ReportSetAside(string, string, connectionport.FailureReason) {}

func openRowsInstallation(t *testing.T) (container.Config, string, *callsHost) {
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
	host := &callsHost{PluginManager: container.NewPluginHost("desktop.editor", plugins)}
	cfg := container.NewConfig(container.Config{SettingsPath: path, PluginHost: host})
	stored, err := cfg.Settings()
	if err != nil {
		t.Fatal(err)
	}
	cfg = cfg.SetSettings(stored)
	t.Cleanup(cfg.UnsubscribeHost)
	return cfg, path, host
}

func TestTheConnectionsListWhereTheirRowsAre(t *testing.T) {
	cfg, _, _ := openRowsInstallation(t)

	entries, err := cfg.OpenConnections().List.Execute(t.Context())

	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || len(entries[0].Uses) != 1 || entries[0].Uses[0].PluginID != rowsPlugin || entries[0].Uses[0].PluginName != "Rows" || entries[0].Uses[0].Operation != "complete" {
		t.Errorf("uses of the first connection = %+v", entries[0].Uses)
	}
}

func TestDeletingAConnectionRemovesTheRowsThatNameItFromTheFile(t *testing.T) {
	cfg, path, _ := openRowsInstallation(t)

	removed, err := cfg.OpenConnections().Delete.Execute(t.Context(), rowsConnection)

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
	cfg, _, _ := openRowsInstallation(t)

	err := cfg.GetSettingsWriter()([]settingsport.Setting{{Path: []string{rowsPlugin, "connections"}, JSON: `[{"connection":"c00000000","model":"m"}]`}})

	if err == nil {
		t.Error("rows that name a connection that does not exist were written")
	}
}

func TestAPluginAsksTheRowsOfItsOwnSettingForAConnection(t *testing.T) {
	cfg, _, host := openRowsInstallation(t)
	cfg.SetModelCalls(noNotices{})

	_, err := host.calls.Complete(t.Context(), rowsPlugin, connectionport.CompleteRequest{})

	var failed *connectionport.AllFailedError
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
	cfg, _, host := openRowsInstallation(t)
	cfg.SetModelCalls(noNotices{})

	_, err := host.calls.Complete(t.Context(), "md.numen.sample.other", connectionport.CompleteRequest{})

	if !errors.Is(err, connectionport.ErrNoConnection) {
		t.Errorf("err = %v, want ErrNoConnection", err)
	}
}
