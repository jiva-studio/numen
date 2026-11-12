package settings

import (
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

var wireContributionKinds = map[plugin.ContributionKind]v1.PluginContributionKind{
	plugin.ContributionCommands:         v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_COMMANDS,
	plugin.ContributionKeybindings:      v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_KEYBINDINGS,
	plugin.ContributionTabKinds:         v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_TAB_KINDS,
	plugin.ContributionReaders:          v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_READERS,
	plugin.ContributionHandlers:         v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_HANDLERS,
	plugin.ContributionTriggers:         v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_TRIGGERS,
	plugin.ContributionTextKinds:        v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_TEXT_KINDS,
	plugin.ContributionNoteTypes:        v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_NOTE_TYPES,
	plugin.ContributionThemes:           v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_THEMES,
	plugin.ContributionSlots:            v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_SLOTS,
	plugin.ContributionListeners:        v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_LISTENERS,
	plugin.ContributionViewCapabilities: v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_VIEW_CAPABILITIES,
	plugin.ContributionDropHandlers:     v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_DROP_HANDLERS,
	plugin.ContributionFrameOrigins:     v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_FRAME_ORIGINS,
	plugin.ContributionConnectionTypes:  v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_CONNECTION_TYPES,
	plugin.ContributionOperations:       v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_OPERATIONS,
	plugin.ContributionPermissions:      v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_PERMISSIONS,
	plugin.ContributionDependencies:     v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_DEPENDENCIES,
	plugin.ContributionServices:         v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_SERVICES,
	plugin.ContributionCapabilities:     v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_CAPABILITIES,
	plugin.ContributionActivationEvents: v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_ACTIVATION_EVENTS,
}

func convertContributionKind(kind plugin.ContributionKind) v1.PluginContributionKind {
	return wireContributionKinds[kind]
}

func newWireContributions(contributions []plugin.Contribution) []*v1.PluginContribution {
	out := make([]*v1.PluginContribution, 0, len(contributions))
	for _, one := range contributions {
		items := make([]*v1.PluginContributionItem, 0, len(one.Items))
		for _, item := range one.Items {
			items = append(items, &v1.PluginContributionItem{Text: item.Text, Detail: item.Detail, Keys: item.Keys})
		}
		out = append(out, &v1.PluginContribution{Kind: convertContributionKind(one.Kind), Items: items})
	}
	return out
}
