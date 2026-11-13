package pluginprocess_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginprocess"
)

func resolveOrchardKeeper(connectionID string) (connection.Resolved, error) {
	if connectionID != "conn-keeper" {
		return connection.Resolved{}, errors.New("no such connection")
	}
	return connection.Resolved{ID: connectionID, Name: "Orchard keeper"}, nil
}

func TestTheHoldersOfAProcessAreNamedByTheirPluginOrConnection(t *testing.T) {
	atlas := plugin.Process{
		PluginID: atlasID, StartTime: getStartTime(), LastUseTime: getStartTime(), LeaseCount: 4,
		StartedBy: plugin.Holder{PluginID: mapsID, Why: "call tool"},
		Holders: []plugin.Holder{
			{Who: plugin.WhoIndexing, Why: "embed notes"},
			{PluginID: mapsID, ConnectionID: "conn-keeper", Why: "call tool"},
			{PluginID: mapsID, ConnectionID: "conn-gone", Why: "call tool"},
			{PluginID: "org.example.unknown", Why: "search"},
		},
	}
	watch := pluginprocess.NewWatchProcesses(newProcesses(atlas), newHealth(), getNames(), resolveOrchardKeeper, clock{})
	watch.Interval = 0

	got := receive(t, watch.Execute(t.Context()))

	if len(got) != 1 {
		t.Fatalf("got %d processes, want one", len(got))
	}
	if want := (plugin.Holder{Who: "Maps", PluginID: mapsID, Why: "call tool"}); got[0].StartedBy != want {
		t.Errorf("started by %+v, want %+v", got[0].StartedBy, want)
	}
	want := []plugin.Holder{
		{Who: plugin.WhoIndexing, Why: "embed notes"},
		{Who: "Orchard keeper", PluginID: mapsID, ConnectionID: "conn-keeper", Why: "call tool"},
		{Who: "Maps", PluginID: mapsID, ConnectionID: "conn-gone", Why: "call tool"},
		{Who: "org.example.unknown", PluginID: "org.example.unknown", Why: "search"},
	}
	if !slices.Equal(got[0].Holders, want) {
		t.Errorf("holders %+v, want %+v", got[0].Holders, want)
	}
}
