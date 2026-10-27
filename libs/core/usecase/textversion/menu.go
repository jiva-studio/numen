package textversion

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/textport"
)

// MenuAction is what a file's menu entry does.
type MenuAction string

const (
	// ActionCreateText asks a plugin for text of the file, or of Version where it is set.
	ActionCreateText MenuAction = "create-text"
	// ActionDeleteVersion deletes a version and its file.
	ActionDeleteVersion MenuAction = "delete-version"
)

// MenuEntry is one entry of a file's menu. Version is the input of a text made
// from a version and the target of a deletion, and empty for a text made from
// the file. PluginName is the name of the plugin that does it, as a person
// reads it. Item is what the plugin declares its entry reads and is drawn
// with, and is empty for a deletion. IsLive says Version is the version shown
// and indexed now.
type MenuEntry struct {
	Action     MenuAction
	PluginID   string
	PluginName string
	Item       textport.MenuItem
	Version    domain.TextVersionID
	IsLive     bool
}

// GetMenu is what a file's menu offers, as the enabled plugins and the file's
// versions say: the entry of every plugin that declares one and takes the
// file's type or the kind of its live version, and a deletion of every
// version, whether or not its plugin is enabled. A plugin that takes both is
// offered the file.
func (u *Versions) GetMenu(ctx context.Context, v domain.Vault, path, fileHash string) ([]MenuEntry, error) {
	versions, err := u.getVersions(ctx, v, path)
	if err != nil {
		return nil, err
	}
	live, hasLive, err := u.findLive(ctx, v, versions, fileHash)
	if err != nil {
		return nil, err
	}

	var entries []MenuEntry
	for _, plugin := range u.plugins.GetTextPlugins() {
		if plugin.Menu.Title == "" {
			continue
		}
		entry := MenuEntry{Action: ActionCreateText, PluginID: plugin.ID, PluginName: plugin.Name, Item: plugin.Menu}
		switch {
		case u.isTaking(ctx, v, plugin, path):
		case hasLive && isKindAccepted(plugin.Kinds, live.Kind):
			entry.Version, entry.IsLive = live.GetID(), true
		default:
			continue
		}
		entries = append(entries, entry)
	}
	for _, version := range versions {
		name := version.PluginID
		if plugin, ok := u.findPlugin(version.PluginID); ok {
			name = plugin.Name
		}
		entries = append(entries, MenuEntry{
			Action: ActionDeleteVersion, PluginID: version.PluginID, PluginName: name, Version: version.GetID(),
			IsLive: hasLive && version.GetID() == live.GetID(),
		})
	}
	return entries, nil
}

// RunMenuEntry does what a menu entry offers.
func (u *Versions) RunMenuEntry(ctx context.Context, v domain.Vault, path string, entry MenuEntry) error {
	switch entry.Action {
	case ActionCreateText:
		return u.CreateText(ctx, v, entry.PluginID, path, entry.Version)
	case ActionDeleteVersion:
		return u.DeleteVersion(ctx, v, path, entry.Version)
	}
	return textport.ErrInvalidVersion
}
