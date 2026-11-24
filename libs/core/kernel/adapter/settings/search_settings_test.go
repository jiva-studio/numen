package settings_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

const searchFile = `{
	"connections": [
		{"id":"c1f0a9e20","type":"md.numen.connections.llm/openrouter","name":"OpenRouter","revision":1}
	],
	"app": {"search": %s}
}`

func openSearch(t *testing.T, search string) settings.File {
	t.Helper()
	cfg, err := openAt(write(t, fmt.Sprintf(searchFile, search)))
	if err != nil {
		t.Fatal(err)
	}
	cfg.DropInvalid(declaredSettings, installedThemes)
	return cfg
}

func TestTheRerankingRowsOfTheShellAreReadAndARowOfAMissingConnectionIsDropped(t *testing.T) {
	cfg := openSearch(t, `{"reranking":[{"connection":"c1f0a9e20","model":"mongodb/rerank-3"},{"connection":"c00000000","model":"x"}]}`)

	if !hasNotice(cfg.Header.Notices, "search.reranking", "row 2") {
		t.Errorf("the notices are %q, want one for the second row", cfg.Header.Notices)
	}
	world := newRowsWorld()
	world.stored = cfg
	setting, err := world.rows().ReadRowSetting(settingsdomain.AppOwner, "llm.rerank")
	if err != nil {
		t.Fatal(err)
	}
	want := []connection.Row{{ConnectionID: workID, Options: map[string]string{"model": "mongodb/rerank-3"}}}
	if !reflect.DeepEqual(setting.Rows, want) || setting.IsDiscovering || !setting.IsDetectedOnly {
		t.Errorf("the rows of the shell are %+v, want %+v, detected only and without general discovery", setting, want)
	}
	if other, err := world.rows().ReadRowSetting(settingsdomain.AppOwner, "llm.embed"); err != nil || len(other.Rows) != 0 {
		t.Errorf("the shell has rows %v for llm.embed, %v", other.Rows, err)
	}
}

func TestTheShellIsAUseOfItsRerankingConnectionAndDeletingTheConnectionRemovesItsRow(t *testing.T) {
	world := newRowsWorld()
	world.stored = openSearch(t, `{"reranking":[{"connection":"c1f0a9e20","model":"mongodb/rerank-3"}]}`)

	uses, err := world.rows().ListUses(workID)
	if err != nil {
		t.Fatal(err)
	}
	if len(uses) == 0 || uses[0].PluginID != settingsdomain.AppOwner || uses[0].PluginName != "Numen" {
		t.Errorf("uses %+v, want the shell first", uses)
	}

	if err := world.rows().RemoveRows(workID); err != nil {
		t.Fatal(err)
	}
	if len(world.writes) == 0 || !reflect.DeepEqual(world.writes[0], settingsdomain.Setting{Path: []string{"app", "search", "reranking"}, JSON: "[]"}) {
		t.Errorf("writes %+v, want the rows of the shell without the connection", world.writes)
	}
	if err := world.rows().Refresh(t.Context(), workID); err != nil {
		t.Fatal(err)
	}
	for _, refreshed := range world.refreshed {
		for _, id := range refreshed {
			if id == settingsdomain.AppOwner {
				t.Error("the shell was sent a configuration as a plugin")
			}
		}
	}
}

func TestAWindowOutOfItsBoundsIsRefusedAndTheDefaultApplies(t *testing.T) {
	for _, window := range []string{"0", "501"} {
		cfg := openSearch(t, `{"rerankWindow":`+window+`,"defaultLimit":30}`)
		limits, stages := cfg.Search.GetLimits()
		if stages.RerankWindow != 60 {
			t.Errorf("a window of %s leaves %d in force, want the default 60", window, stages.RerankWindow)
		}
		if !hasNotice(cfg.Header.Notices, "search.rerankWindow") {
			t.Errorf("a window of %s gives the notices %q", window, cfg.Header.Notices)
		}
		if limits.DefaultLimit != 30 {
			t.Errorf("the default limit in force is %d, want 30", limits.DefaultLimit)
		}
	}
}
