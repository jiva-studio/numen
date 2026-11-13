package plugin

import (
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// ContributionKind is a section of a manifest whose entries the plugin page lists.
type ContributionKind int

// The sections of a manifest, in the order the plugin page lists them.
const (
	ContributionCommands ContributionKind = iota + 1
	ContributionKeybindings
	ContributionTabKinds
	ContributionReaders
	ContributionHandlers
	ContributionTriggers
	ContributionTextKinds
	ContributionNoteTypes
	ContributionThemes
	ContributionSlots
	ContributionListeners
	ContributionViewCapabilities
	ContributionDropHandlers
	ContributionFrameOrigins
	ContributionConnectionTypes
	ContributionOperations
	ContributionPermissions
	ContributionDependencies
	ContributionServices
	ContributionCapabilities
	ContributionActivationEvents
)

// Contribution is one section of a manifest and its entries, in the order the manifest declares
// them.
type Contribution struct {
	Kind  ContributionKind
	Items []ContributionItem
}

// ContributionItem is one entry of a section. Text is its title, or its id where it declares none;
// Detail is its id or target as the manifest writes it; Keys is the shortcut of a command or a
// keybinding, such as mod+shift+a.
type ContributionItem struct {
	Text   string
	Detail string
	Keys   string
}

// ListContributions returns the sections of the manifest that declare anything, in the order of
// ContributionKind. A keybinding of a command the manifest declares is the Keys of that command and
// is not listed under ContributionKeybindings.
func ListContributions(m manifest.Manifest) []Contribution {
	sections := append(listContributesSections(m.Contributes), listManifestSections(m)...)
	declared := make([]Contribution, 0, len(sections))
	for _, one := range sections {
		if len(one.Items) > 0 {
			declared = append(declared, one)
		}
	}
	return declared
}

func listContributesSections(c manifest.Contributes) []Contribution {
	return []Contribution{
		{ContributionCommands, listCommands(c)},
		{ContributionKeybindings, listForeignKeybindings(c)},
		{ContributionTabKinds, mapItems(c.TabKinds, func(one manifest.TabKind) ContributionItem {
			return ContributionItem{Text: getTitle(one.Title, one.Kind), Detail: one.Kind}
		})},
		{ContributionReaders, mapItems(c.Readers, func(one manifest.Reader) ContributionItem {
			return ContributionItem{Text: one.ID, Detail: joinContentTypes(one)}
		})},
		{ContributionHandlers, mapItems(c.Handlers, func(one manifest.Handler) ContributionItem {
			return ContributionItem{Text: getTitle(one.Title, one.Command), Detail: one.Command}
		})},
		{ContributionTriggers, mapItems(c.Triggers, func(one manifest.Trigger) ContributionItem {
			return ContributionItem{Text: one.Event, Detail: one.Command}
		})},
		{ContributionTextKinds, mapItems(c.TextKinds, func(one manifest.TextKind) ContributionItem {
			return ContributionItem{Text: getTitle(one.Title, one.Kind), Detail: one.Kind}
		})},
		{ContributionNoteTypes, mapItems(c.NoteTypes, func(one manifest.NoteType) ContributionItem {
			return ContributionItem{Text: one.Type}
		})},
		{ContributionThemes, mapItems(c.Themes, func(one manifest.Theme) ContributionItem {
			return ContributionItem{Text: getTitle(one.Name, one.ID), Detail: one.ID}
		})},
		{ContributionSlots, mapItems(c.Slots, func(one manifest.SlotEntry) ContributionItem {
			return ContributionItem{Text: one.Slot, Detail: getSlotTarget(one)}
		})},
		{ContributionListeners, mapItems(c.Listeners, func(one manifest.Listener) ContributionItem {
			return ContributionItem{Text: one.Name, Detail: getListenerPort(one)}
		})},
		{ContributionViewCapabilities, mapItems(c.ViewCapabilities, newNameItem)},
		{ContributionDropHandlers, mapItems(c.DropHandlers, func(one manifest.DropHandler) ContributionItem {
			return ContributionItem{Text: strings.Join(one.Types, ", ")}
		})},
		{ContributionFrameOrigins, mapItems(c.FrameOrigins, newNameItem)},
		{ContributionConnectionTypes, mapItems(c.ConnectionTypes, func(one manifest.ConnectionType) ContributionItem {
			return ContributionItem{Text: getTitle(one.Title, one.ID), Detail: one.ID}
		})},
		{ContributionOperations, mapItems(c.Operations, func(one manifest.Operation) ContributionItem {
			return ContributionItem{Text: getTitle(one.Title, one.Name), Detail: one.Name}
		})},
	}
}

func listManifestSections(m manifest.Manifest) []Contribution {
	return []Contribution{
		{ContributionPermissions, mapItems(m.Permissions, newNameItem)},
		{ContributionDependencies, mapItems(m.Dependencies, func(one manifest.DependencyRequirement) ContributionItem {
			return ContributionItem{Text: one.ID, Detail: string(one.Kind)}
		})},
		{ContributionServices, mapItems(m.Services, newNameItem)},
		{ContributionCapabilities, mapItems(m.Capabilities, newNameItem)},
		{ContributionActivationEvents, mapItems(m.ActivationEvents, newNameItem)},
	}
}

func listCommands(c manifest.Contributes) []ContributionItem {
	keys := map[string]string{}
	for _, one := range c.Keybindings {
		if _, isBound := keys[one.Command]; !isBound {
			keys[one.Command] = one.Key
		}
	}
	return mapItems(c.Commands, func(one manifest.Command) ContributionItem {
		return ContributionItem{Text: getTitle(one.Title, one.ID), Detail: one.ID, Keys: keys[one.ID]}
	})
}

func listForeignKeybindings(c manifest.Contributes) []ContributionItem {
	declared := map[string]bool{}
	for _, one := range c.Commands {
		declared[one.ID] = true
	}
	var items []ContributionItem
	for _, one := range c.Keybindings {
		if !declared[one.Command] {
			items = append(items, ContributionItem{Text: one.Command, Keys: one.Key})
		}
	}
	return items
}

func mapItems[T any](entries []T, toItem func(T) ContributionItem) []ContributionItem {
	items := make([]ContributionItem, 0, len(entries))
	for _, one := range entries {
		items = append(items, toItem(one))
	}
	return items
}

func newNameItem(name string) ContributionItem {
	return ContributionItem{Text: name}
}

func getTitle(title, id string) string {
	if title != "" {
		return title
	}
	return id
}

func getSlotTarget(entry manifest.SlotEntry) string {
	for _, target := range []string{entry.Command, entry.View, entry.Submenu} {
		if target != "" {
			return target
		}
	}
	return ""
}

func getListenerPort(listener manifest.Listener) string {
	if listener.PortSetting != "" {
		return listener.PortSetting
	}
	if listener.Port != 0 {
		return strconv.Itoa(listener.Port)
	}
	return ""
}

func joinContentTypes(reader manifest.Reader) string {
	types := make([]string, 0, len(reader.ContentTypes))
	for _, one := range reader.ContentTypes {
		types = append(types, string(one))
	}
	return strings.Join(types, ", ")
}
