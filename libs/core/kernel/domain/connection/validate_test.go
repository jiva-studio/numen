package connection_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func newConnection(id, name string) connection.Connection {
	return connection.Connection{ID: id, TypeID: "md.numen.connections.llm/openrouter", Name: name, Revision: 1}
}

func TestFilterValidRejectsEachInvalidConnectionWithItsReason(t *testing.T) {
	badType := newConnection("c0000000c", "Local")
	badType.TypeID = "openrouter"
	badRevision := newConnection("c0000000d", "Office")
	badRevision.Revision = 0
	list := []connection.Connection{
		newConnection("c0000000a", "OpenRouter"),
		newConnection("c0000000a", "Second"),
		newConnection("c0000000b", "openrouter"),
		newConnection("bad", "Third"),
		badType,
		badRevision,
	}

	kept, rejections := connection.FilterValid(list)

	if len(kept) != 1 || kept[0].Name != "OpenRouter" {
		t.Errorf("kept = %+v, want the first connection only", kept)
	}
	want := []connection.Rejection{
		{ID: "c0000000a", Reason: "the id is used twice"},
		{ID: "c0000000b", Reason: "the name is used twice"},
		{ID: "bad", Reason: "the id has the wrong shape"},
		{ID: "c0000000c", Reason: "the type has the wrong shape"},
		{ID: "c0000000d", Reason: "the revision is below 1"},
	}
	if !reflect.DeepEqual(rejections, want) {
		t.Errorf("rejections = %+v, want %+v", rejections, want)
	}
}

func TestListGrantsReturnsTheOperationsEachPluginIsGrantedInPluginOrder(t *testing.T) {
	plugins := []plugin.Info{
		{ID: "md.numen.agents.lantern", Name: "Lantern agent", Permissions: []string{
			"vault:read", manifest.FormatConnectionsUse("llm.chat"), manifest.FormatConnectionsUse("llm.profile"),
		}},
		{ID: "md.numen.views.outline", Name: "Outline", Permissions: []string{"vault:read"}},
	}

	grants := connection.ListGrants(plugins)

	want := []connection.Grant{{PluginID: "md.numen.agents.lantern", PluginName: "Lantern agent", Operations: []string{"llm.chat", "llm.profile"}}}
	if !reflect.DeepEqual(grants, want) {
		t.Errorf("grants = %+v, want %+v", grants, want)
	}
}
