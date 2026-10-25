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
	// ActionDeleteVersion asks the plugin that made a version to delete it.
	ActionDeleteVersion MenuAction = "delete-version"
)

// MenuEntry is one entry of a file's menu. Version is the input of a
// proofreading and the target of a deletion, and empty for a recognition.
type MenuEntry struct {
	Action   MenuAction
	PluginID string
	Version  domain.TextVersionID
}

// GetMenu is what a file's menu offers, as the enabled plugins and the file's
// versions say: a recognition by every plugin that reads the file's type, a
// proofreading of the live version by every plugin that reads its format, and
// a deletion of every version whose plugin is enabled.
func (u *Versions) GetMenu(ctx context.Context, v domain.Vault, path, fileHash string) ([]MenuEntry, error) {
	versions, err := u.getVersions(ctx, v, path)
	if err != nil {
		return nil, err
	}
	plugins := u.plugins.GetTextPlugins()
	live, hasLive := domain.FindLiveVersion(versions, fileHash, u.isEnabled)

	var entries []MenuEntry
	for _, plugin := range plugins {
		if plugin.Files.IsFileAccepted(path) {
			entries = append(entries, MenuEntry{Action: ActionRecognise, PluginID: plugin.ID})
		}
	}
	for _, plugin := range plugins {
		if hasLive && slices.Contains(plugin.Formats, live.Format) {
			entries = append(entries, MenuEntry{Action: ActionProofread, PluginID: plugin.ID, Version: live.GetID()})
		}
	}
	for _, version := range versions {
		if u.isEnabled(version.PluginID) {
			entries = append(entries, MenuEntry{Action: ActionDeleteVersion, PluginID: version.PluginID, Version: version.GetID()})
		}
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
