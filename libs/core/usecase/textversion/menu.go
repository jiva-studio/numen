package textversion

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/textport"
)

// MenuAction is what a file's menu entry does.
type MenuAction string

const (
	// ActionRecognise asks a plugin that reads the file's type for text of the file.
	ActionRecognise MenuAction = "recognise"
	// ActionProofread asks a plugin that reads text for a version made from the live one.
	ActionProofread MenuAction = "proofread"
	// ActionDeleteVersion deletes a version and its file.
	ActionDeleteVersion MenuAction = "delete-version"
)

// MenuEntry is one entry of a file's menu. Version is the input of a
// proofreading and the target of a deletion, and empty for a recognition.
// PluginName is the name of the plugin that does it, as a person reads it.
// IsLive says Version is the version shown and indexed now.
type MenuEntry struct {
	Action     MenuAction
	PluginID   string
	PluginName string
	Version    domain.TextVersionID
	IsLive     bool
}

// GetMenu is what a file's menu offers, as the enabled plugins and the file's
// versions say: a recognition by every plugin that reads the file's type, a
// proofreading of the live version by every plugin that reads its kind, and a
// deletion of every version, whether or not its plugin is enabled.
func (u *Versions) GetMenu(ctx context.Context, v domain.Vault, path, fileHash string) ([]MenuEntry, error) {
	versions, err := u.getVersions(ctx, v, path)
	if err != nil {
		return nil, err
	}
	plugins := u.plugins.GetTextPlugins()
	live, hasLive, err := u.findLive(ctx, v, versions, fileHash)
	if err != nil {
		return nil, err
	}

	var entries []MenuEntry
	for _, plugin := range plugins {
		if plugin.Files.IsFileAccepted(path) {
			entries = append(entries, MenuEntry{Action: ActionRecognise, PluginID: plugin.ID, PluginName: plugin.Name})
		}
	}
	for _, plugin := range plugins {
		if hasLive && slices.Contains(plugin.Kinds, live.Kind) {
			entries = append(entries, MenuEntry{Action: ActionProofread, PluginID: plugin.ID, PluginName: plugin.Name, Version: live.GetID(), IsLive: true})
		}
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
	case ActionRecognise:
		return u.CreateText(ctx, v, entry.PluginID, path, "")
	case ActionProofread:
		return u.CreateText(ctx, v, entry.PluginID, path, entry.Version)
	case ActionDeleteVersion:
		return u.DeleteVersion(ctx, v, path, entry.Version)
	}
	return textport.ErrInvalidVersion
}
