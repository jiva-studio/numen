package settings

import (
	"testing"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func TestEveryContributionKindIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.PluginContributionKind]plugin.ContributionKind{
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_COMMANDS:          plugin.ContributionCommands,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_KEYBINDINGS:       plugin.ContributionKeybindings,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_TAB_KINDS:         plugin.ContributionTabKinds,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_READERS:           plugin.ContributionReaders,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_HANDLERS:          plugin.ContributionHandlers,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_TRIGGERS:          plugin.ContributionTriggers,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_TEXT_KINDS:        plugin.ContributionTextKinds,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_NOTE_TYPES:        plugin.ContributionNoteTypes,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_THEMES:            plugin.ContributionThemes,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_SLOTS:             plugin.ContributionSlots,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_LISTENERS:         plugin.ContributionListeners,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_VIEW_CAPABILITIES: plugin.ContributionViewCapabilities,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_DROP_HANDLERS:     plugin.ContributionDropHandlers,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_FRAME_ORIGINS:     plugin.ContributionFrameOrigins,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_CONNECTION_TYPES:  plugin.ContributionConnectionTypes,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_OPERATIONS:        plugin.ContributionOperations,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_PERMISSIONS:       plugin.ContributionPermissions,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_DEPENDENCIES:      plugin.ContributionDependencies,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_SERVICES:          plugin.ContributionServices,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_CAPABILITIES:      plugin.ContributionCapabilities,
		v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_ACTIVATION_EVENTS: plugin.ContributionActivationEvents,
	}, convertContributionKind)
}

func TestNewWirePluginInfos_CarriesTheContributions(t *testing.T) {
	wire := NewWirePluginInfos([]plugin.Info{{
		ID: "a.b",
		Contributions: []plugin.Contribution{{
			Kind:  plugin.ContributionCommands,
			Items: []plugin.ContributionItem{{Text: "Open", Detail: "a.b.open", Keys: "mod+o"}},
		}},
	}})

	got := wire[0].GetContributions()
	if len(got) != 1 || got[0].GetKind() != v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_COMMANDS {
		t.Fatalf("the contributions are %v", got)
	}
	item := got[0].GetItems()[0]
	if item.GetText() != "Open" || item.GetDetail() != "a.b.open" || item.GetKeys() != "mod+o" {
		t.Errorf("the item is %v", item)
	}
}
