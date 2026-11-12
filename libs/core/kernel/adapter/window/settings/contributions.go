package settings

import (
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func convertContributionKind(kind plugin.ContributionKind) v1.PluginContributionKind {
	switch kind {
	case plugin.ContributionCommands:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_COMMANDS
	case plugin.ContributionKeybindings:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_KEYBINDINGS
	case plugin.ContributionTabKinds:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_TAB_KINDS
	case plugin.ContributionReaders:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_READERS
	case plugin.ContributionHandlers:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_HANDLERS
	case plugin.ContributionTriggers:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_TRIGGERS
	case plugin.ContributionTextKinds:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_TEXT_KINDS
	case plugin.ContributionNoteTypes:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_NOTE_TYPES
	case plugin.ContributionThemes:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_THEMES
	case plugin.ContributionSlots:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_SLOTS
	case plugin.ContributionListeners:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_LISTENERS
	case plugin.ContributionViewCapabilities:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_VIEW_CAPABILITIES
	case plugin.ContributionDropHandlers:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_DROP_HANDLERS
	case plugin.ContributionFrameOrigins:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_FRAME_ORIGINS
	case plugin.ContributionConnectionTypes:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_CONNECTION_TYPES
	case plugin.ContributionOperations:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_OPERATIONS
	case plugin.ContributionPermissions:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_PERMISSIONS
	case plugin.ContributionDependencies:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_DEPENDENCIES
	case plugin.ContributionServices:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_SERVICES
	case plugin.ContributionCapabilities:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_CAPABILITIES
	case plugin.ContributionActivationEvents:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_ACTIVATION_EVENTS
	default:
		return v1.PluginContributionKind_PLUGIN_CONTRIBUTION_KIND_UNSPECIFIED
	}
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
