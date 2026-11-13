package settings_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const (
	embeddingsID = "md.numen.engines.embeddings"
	workID       = "c1f0a9e20"
	homeID       = "c77b01d40"
)

type fixedConnections []connection.Connection

func (c fixedConnections) ListConnections() ([]connection.Connection, error) { return c, nil }

func (c fixedConnections) ChangeConnections(func([]connection.Connection) ([]connection.Connection, error)) error {
	return errors.New("not used")
}

// fixedDetection has detected Kite agent of the type kite, and the provider of the type heron has not answered yet.
type fixedDetection struct{}

const (
	kiteAgentID  = "org.example.agents/kite/Kite agent"
	heronTypeID  = "org.example.agents/heron"
	heronAgentID = heronTypeID + "/Heron agent"
	wrenAgentID  = "org.example.gone/wren/Wren agent"
)

func (fixedDetection) ListDetected() []connection.Connection {
	return []connection.Connection{{ID: kiteAgentID, TypeID: "org.example.agents/kite", Name: "Kite agent"}}
}

func (fixedDetection) IsPending(typeID string) bool { return typeID == heronTypeID }

func (fixedDetection) Detect(context.Context, string) {}

func (fixedDetection) Refresh() {}

func rowsSetting(key, operation string, isShown bool, value string) plugin.Setting {
	return plugin.Setting{Key: key, Type: "connections", Operations: []string{operation}, IsShown: isShown, ValueJSON: value}
}

func rowsPlugin(id, name string, rows ...plugin.Setting) plugin.Info {
	return plugin.Info{ID: id, Name: name, Settings: append([]plugin.Setting{{Key: "label", Type: "string"}}, rows...)}
}

type rowsWorld struct {
	plugins   []plugin.Info
	stored    settings.File
	writes    []settingsdomain.Setting
	failWrite error
	refreshed [][]string
	failSend  error
	reported  []error
}

func (w *rowsWorld) rows() *settings.ConnectionRows {
	return settings.NewConnectionRows(settings.ConnectionRowsSource{
		Connections: fixedConnections{{ID: workID, Name: "Work"}, {ID: homeID, Name: "Ollama"}},
		Detected:    fixedDetection{},
		Plugins:     func() []plugin.Info { return w.plugins },
		Declared:    declaredWithRows,
		ReadStored:  func() (settings.File, error) { return w.stored, nil },
		Write: func(entries []settingsdomain.Setting) error {
			w.writes = append(w.writes, entries...)
			return w.failWrite
		},
		Refresh: func(_ context.Context, forced ...string) error {
			w.refreshed = append(w.refreshed, forced)
			return w.failSend
		},
		Report: func(err error) { w.reported = append(w.reported, err) },
	})
}

func newRowsWorld() *rowsWorld {
	return &rowsWorld{plugins: []plugin.Info{
		rowsPlugin(embeddingsID, "Embeddings",
			rowsSetting("connections", "llm.embed", true, `[{"connection":"`+workID+`","model":"text-embedding-3-small"}]`)),
		rowsPlugin(rowsPluginID, "Proofreading",
			rowsSetting("connections", "llm.complete", true,
				`[{"connection":"`+workID+`","model":"google/gemini-2.5-flash","reasoning":"low"},{"connection":"`+homeID+`","model":"qwen2.5:14b"}]`)),
	}}
}

func TestReadRowSettingReturnsTheRowsOfThePluginInTheOrderOfUse(t *testing.T) {
	world := newRowsWorld()

	rows, err := readRows(world, rowsPluginID, "llm.complete")

	if err != nil {
		t.Fatal(err)
	}
	want := []connection.Row{
		{ConnectionID: workID, Options: map[string]string{"model": "google/gemini-2.5-flash", "reasoning": "low"}},
		{ConnectionID: homeID, Options: map[string]string{"model": "qwen2.5:14b"}},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %v, want %v", rows, want)
	}
}

func TestReadRowSettingAnswersOnlyForTheOperationOfTheSetting(t *testing.T) {
	world := newRowsWorld()

	rows, err := readRows(world, rowsPluginID, "llm.embed")

	if err != nil || len(rows) != 0 {
		t.Errorf("rows = %v, %v, want none: proofreading has no setting for llm.embed", rows, err)
	}
}

func TestReadRowSettingAnswersNothingForAPluginThatHasNoSuchSetting(t *testing.T) {
	world := newRowsWorld()

	rows, err := readRows(world, "md.numen.nobody", "llm.complete")

	if err != nil || len(rows) != 0 {
		t.Errorf("rows = %v, %v, want none", rows, err)
	}
}

func TestReadRowSettingReturnsNoRowForAHiddenSetting(t *testing.T) {
	world := newRowsWorld()
	world.plugins[0] = rowsPlugin(embeddingsID, "Embeddings",
		rowsSetting("connections", "llm.embed", false, `[{"connection":"`+workID+`","model":"m"}]`))

	rows, err := readRows(world, embeddingsID, "llm.embed")

	if err != nil || len(rows) != 0 {
		t.Errorf("rows = %v, %v: the rows of a local-model source must not be used", rows, err)
	}
}

func TestReadRowSettingReturnsOnlyRowsOfExistingConnections(t *testing.T) {
	world := newRowsWorld()
	world.plugins[1] = rowsPlugin(rowsPluginID, "Proofreading", rowsSetting("connections", "llm.complete", true,
		`[{"connection":"c00000000","model":"a"},{"connection":"`+homeID+`","model":"b"}]`))

	rows, err := readRows(world, rowsPluginID, "llm.complete")

	if err != nil || len(rows) != 1 || rows[0].ConnectionID != homeID {
		t.Errorf("rows = %v, %v, want the row of the connection that exists", rows, err)
	}
}

func TestReadRowSettingKeepsTheRowsOfADetectedConnectionAndOfATypeNotAnsweredYet(t *testing.T) {
	world := newRowsWorld()
	world.plugins[1] = rowsPlugin(rowsPluginID, "Proofreading", rowsSetting("connections", "llm.complete", true,
		`[{"connection":"`+wrenAgentID+`"},{"connection":"`+kiteAgentID+`","model":"kite-small"},{"connection":"`+heronAgentID+`"}]`))

	rows, err := readRows(world, rowsPluginID, "llm.complete")

	if err != nil || len(rows) != 2 || rows[0].ConnectionID != kiteAgentID || rows[0].Options["model"] != "kite-small" || rows[1].ConnectionID != heronAgentID {
		t.Errorf("rows = %v, %v, want the rows of Kite agent and Heron agent", rows, err)
	}
}

func TestReadRowSettingIgnoresAValueThatIsNoListOfRows(t *testing.T) {
	world := newRowsWorld()
	world.plugins[1] = rowsPlugin(rowsPluginID, "Proofreading", rowsSetting("connections", "llm.complete", true, `"c1f0a9e20"`))

	rows, err := readRows(world, rowsPluginID, "llm.complete")

	if err != nil || len(rows) != 0 {
		t.Errorf("rows = %v, %v, want none", rows, err)
	}
}

func TestListUsesNamesEachSettingOfRowsThatNamesTheConnection(t *testing.T) {
	world := newRowsWorld()

	uses, err := world.rows().ListUses(workID)

	if err != nil {
		t.Fatal(err)
	}
	want := []connection.Use{
		{PluginID: embeddingsID, PluginName: "Embeddings", Operations: []string{"llm.embed"}},
		{PluginID: rowsPluginID, PluginName: "Proofreading", Operations: []string{"llm.complete"}},
	}
	if !reflect.DeepEqual(uses, want) {
		t.Errorf("uses = %v, want %v", uses, want)
	}
}

func TestListUsesNamesASettingOnceWhateverTheNumberOfItsRowsForTheConnection(t *testing.T) {
	world := newRowsWorld()
	world.plugins[1] = rowsPlugin(rowsPluginID, "Proofreading", rowsSetting("connections", "llm.complete", true,
		`[{"connection":"`+homeID+`","model":"a"},{"connection":"`+homeID+`","model":"b"}]`))

	uses, err := world.rows().ListUses(homeID)

	if err != nil || len(uses) != 1 {
		t.Errorf("uses = %v, %v, want one", uses, err)
	}
}

func TestListUsesReturnsNoUseForAHiddenSetting(t *testing.T) {
	world := newRowsWorld()
	world.plugins[0] = rowsPlugin(embeddingsID, "Embeddings",
		rowsSetting("connections", "llm.embed", false, `[{"connection":"`+workID+`","model":"m"}]`))

	uses, err := world.rows().ListUses(workID)

	if err != nil || len(uses) != 1 || uses[0].PluginID != rowsPluginID {
		t.Errorf("uses = %v, %v, want only proofreading", uses, err)
	}
}

func readStoredFile(t *testing.T, body string) settings.File {
	t.Helper()
	cfg, err := openAt(write(t, body))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestRemoveRowsWritesEverySettingOfRowsWithoutTheRowsOfTheConnection(t *testing.T) {
	world := newRowsWorld()
	world.stored = readStoredFile(t, `{
		"md.numen.engines.embeddings": {"connections": [{"connection":"c1f0a9e20","model":"text-embedding-3-small"}]},
		"md.numen.tools.proofreading": {"label":"x","connections": [
			{"connection":"c1f0a9e20","model":"google/gemini-2.5-flash","reasoning":"low"},
			{"connection":"c77b01d40","model":"qwen2.5:14b"},
			{"connection":"c1f0a9e20","model":"other"}]}
	}`)

	if err := world.rows().RemoveRows(workID); err != nil {
		t.Fatal(err)
	}

	got := map[string]string{}
	for _, one := range world.writes {
		got[strings.Join(one.Path, ".")] = one.JSON
	}
	if len(got) != 2 {
		t.Fatalf("writes = %v, want the two settings that named the connection", world.writes)
	}
	if got[embeddingsID+".connections"] != `[]` {
		t.Errorf("embeddings = %s", got[embeddingsID+".connections"])
	}
	if got[rowsPluginID+".connections"] != `[{"connection":"c77b01d40","model":"qwen2.5:14b"}]` {
		t.Errorf("proofreading = %s", got[rowsPluginID+".connections"])
	}
}

func TestRemoveRowsWritesNothingWhenNoRowNamesTheConnection(t *testing.T) {
	world := newRowsWorld()
	world.stored = readStoredFile(t, `{"md.numen.tools.proofreading": {"connections": [{"connection":"c77b01d40","model":"m"}]}}`)

	if err := world.rows().RemoveRows(workID); err != nil {
		t.Fatal(err)
	}

	if len(world.writes) != 0 {
		t.Errorf("writes = %v", world.writes)
	}
}

func TestRemoveRowsReachesRowsOfASettingThatIsNotShown(t *testing.T) {
	world := newRowsWorld()
	world.stored = readStoredFile(t, `{"md.numen.engines.embeddings": {"source":"local","connections": [{"connection":"c1f0a9e20","model":"m"}]}}`)
	world.plugins[0] = rowsPlugin(embeddingsID, "Embeddings", rowsSetting("connections", "llm.embed", false, ""))

	if err := world.rows().RemoveRows(workID); err != nil {
		t.Fatal(err)
	}

	if len(world.writes) != 1 || world.writes[0].JSON != `[]` {
		t.Errorf("writes = %v, want the rows of the hidden setting emptied", world.writes)
	}
}

func TestRemoveRowsReportsAWriteThatFails(t *testing.T) {
	world := newRowsWorld()
	world.stored = readStoredFile(t, `{"md.numen.tools.proofreading": {"connections": [{"connection":"c1f0a9e20","model":"m"}]}}`)
	world.failWrite = errors.New("the disk is full")

	err := world.rows().RemoveRows(workID)

	if !errors.Is(err, world.failWrite) {
		t.Errorf("err = %v, want the failure to write", err)
	}
}

func TestRefreshSendsTheConfigurationAgainToThePluginsWhoseRowsNameTheConnection(t *testing.T) {
	world := newRowsWorld()

	if err := world.rows().Refresh(t.Context(), homeID); err != nil {
		t.Fatal(err)
	}

	if len(world.refreshed) != 1 || !slices.Equal(world.refreshed[0], []string{rowsPluginID}) {
		t.Errorf("refreshed = %v, want proofreading only", world.refreshed)
	}
}

func TestRefreshSendsNothingWhenNoPluginNamesTheConnection(t *testing.T) {
	world := newRowsWorld()

	if err := world.rows().Refresh(t.Context(), "c00000000"); err != nil {
		t.Fatal(err)
	}

	if len(world.refreshed) != 0 {
		t.Errorf("refreshed = %v", world.refreshed)
	}
}

func TestRefreshReportsAPluginThatCannotBeConfiguredAndGoesOn(t *testing.T) {
	world := newRowsWorld()
	world.failSend = errors.New("embeddings did not answer")

	err := world.rows().Refresh(t.Context(), workID)

	if err != nil {
		t.Errorf("err = %v, want none: the plugin is configured on its next start", err)
	}
	if len(world.reported) != 1 || !strings.Contains(world.reported[0].Error(), "embeddings did not answer") {
		t.Errorf("reported = %v", world.reported)
	}
}

func TestRefreshStopsWhenTheContextEnds(t *testing.T) {
	world := newRowsWorld()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := world.rows().Refresh(ctx, workID)

	if !errors.Is(err, context.Canceled) || len(world.refreshed) != 0 {
		t.Errorf("err = %v, refreshed = %v", err, world.refreshed)
	}
}

func readRows(world *rowsWorld, pluginID, operation string) ([]connection.Row, error) {
	setting, err := world.rows().ReadRowSetting(pluginID, operation)
	return setting.Rows, err
}

func TestReadRowSettingReturnsWhetherTheSettingDiscovers(t *testing.T) {
	world := newRowsWorld()
	discovering := rowsSetting("connections", "acp.task", true, "")
	discovering.IsDiscovering = true
	world.plugins = append(world.plugins, rowsPlugin("md.numen.views.agent.chat", "Agent Chat", discovering))

	setting, err := world.rows().ReadRowSetting("md.numen.views.agent.chat", "acp.task")

	if err != nil || !setting.IsDiscovering || len(setting.Rows) != 0 {
		t.Errorf("setting = %+v, %v, want no row and discovery", setting, err)
	}
}

func TestReadRowSettingDoesNotDiscoverForAHiddenSetting(t *testing.T) {
	world := newRowsWorld()
	hidden := rowsSetting("connections", "acp.task", false, "")
	hidden.IsDiscovering = true
	world.plugins = append(world.plugins, rowsPlugin("md.numen.views.agent.chat", "Agent Chat", hidden))

	setting, err := world.rows().ReadRowSetting("md.numen.views.agent.chat", "acp.task")

	if err != nil || setting.IsDiscovering {
		t.Errorf("setting = %+v, %v, want no discovery", setting, err)
	}
}

func TestReadRowSettingDiscoversForAPluginThatUsesConnectionsAndHasNoSettingForTheOperation(t *testing.T) {
	world := newRowsWorld()
	agent := rowsPlugin("md.numen.agents.lantern", "Lantern agent")
	agent.Permissions = []string{manifest.PermissionConnectionsUse}
	world.plugins = append(world.plugins, agent)

	setting, err := world.rows().ReadRowSetting("md.numen.agents.lantern", "llm.chat")

	if err != nil || !setting.IsDiscovering || len(setting.Rows) != 0 {
		t.Errorf("setting = %+v, %v, want no row and discovery", setting, err)
	}
}

func TestReadRowSettingDoesNotDiscoverForAPluginWithoutThePermissionToUseConnections(t *testing.T) {
	world := newRowsWorld()
	world.plugins = append(world.plugins, rowsPlugin("md.numen.agents.lantern", "Lantern agent"))

	setting, err := world.rows().ReadRowSetting("md.numen.agents.lantern", "llm.chat")

	if err != nil || setting.IsDiscovering {
		t.Errorf("setting = %+v, %v, want no discovery", setting, err)
	}
}

func TestReadRowSettingReadsTheRowsOfTheSettingOfAPluginThatUsesConnections(t *testing.T) {
	world := newRowsWorld()
	world.plugins[1].Permissions = []string{manifest.PermissionConnectionsUse}

	setting, err := world.rows().ReadRowSetting(rowsPluginID, "llm.complete")

	if err != nil || setting.IsDiscovering || len(setting.Rows) != 2 {
		t.Errorf("setting = %+v, %v, want the two rows of the setting and no discovery", setting, err)
	}
}
