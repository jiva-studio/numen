package textversion

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
)

// MenuEntry is one entry of a file's menu: a command a plugin declares, offered
// over the file or over a version of it. Title and Icon are the command's
// declaration; Icon is a Lucide icon name. Version is the text version the
// command runs over, and is empty for a command over the file. IsLive marks
// Version as the version displayed and indexed now.
type MenuEntry struct {
	PluginID   string
	PluginName string
	Command    string
	Title      string
	Icon       string
	Version    domain.TextVersionID
	IsLive     bool
}

// FileMenu is the menu of a vault file: the commands the enabled plugins
// declare that apply to the file as it is now. A command that names content
// types applies to a file of one of them; a command with conditions applies
// to each version of the file in whose context they all hold.
type FileMenu struct {
	versions *Versions
	commands pluginport.PluginCommands
	types    contenttypeport.ContentTypeTable
}

// NewFileMenu is the menu the plugin commands make of the vault files, where the
// table gives the type of a file.
func NewFileMenu(versions *Versions, commands pluginport.PluginCommands, types contenttypeport.ContentTypeTable) (*FileMenu, error) {
	if versions == nil || commands == nil || types == nil {
		return nil, errors.New("a file menu needs text versions, plugin commands and a content type table")
	}
	return &FileMenu{versions: versions, commands: commands, types: types}, nil
}

// GetMenu is what the file's menu offers. A file is read and hashed only when
// it has versions, to tell which is live.
func (u *FileMenu) GetMenu(ctx context.Context, v domain.Vault, path string) ([]MenuEntry, error) {
	versions, err := u.versions.getVersions(ctx, v, path)
	if err != nil {
		return nil, err
	}
	live, hasLive, err := u.findLiveVersion(ctx, v, path, versions)
	if err != nil {
		return nil, err
	}
	contentType := u.types.GetContentType(path)
	var entries []MenuEntry
	for _, one := range u.commands.ListCommands() {
		command := one.Command
		entry := MenuEntry{PluginID: one.PluginID, PluginName: one.PluginName, Command: command.ID, Title: command.Title, Icon: command.Icon}
		switch {
		case len(command.When) > 0:
			for _, version := range listCandidates(command, versions, live, hasLive) {
				entry.Version, entry.IsLive = version.GetID(), hasLive && version.GetID() == live.GetID()
				entries = append(entries, entry)
			}
		case len(command.ContentTypes) > 0 && len(command.NoteTypes) == 0 && !hasRequiredArgument(command) && command.CanRunOn(contentType, ""):
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

// findLiveVersion is the live version of the file among its versions. The file is read and hashed
// only when it has versions.
func (u *FileMenu) findLiveVersion(ctx context.Context, v domain.Vault, path string, versions []domain.TextVersion) (domain.TextVersion, bool, error) {
	if len(versions) == 0 {
		return domain.TextVersion{}, false, nil
	}
	fileHash, err := u.versions.GetFileHash(ctx, v, path)
	if err != nil {
		return domain.TextVersion{}, false, fmt.Errorf("read %s: %w", path, err)
	}
	return u.versions.findLive(ctx, v, versions, fileHash)
}

// hasRequiredArgument reports whether the command cannot run without a value a menu does not give.
func hasRequiredArgument(command pluginport.PluginCommand) bool {
	for _, argument := range command.Arguments {
		if argument.IsRequired {
			return true
		}
	}
	return false
}

// The keys of the context the menu supplies for each version of a file. A condition of a command
// names them, and every condition has to hold in the context of one version.
const (
	// ContextKeyLive is "true" for the version displayed and indexed now, and "false" for the others.
	ContextKeyLive = "version.live"
	// ContextKeyKind is the kind of the version, as the plugin that made it names it.
	ContextKeyKind = "version.kind"
	// ContextKeyPlugin is the id of the plugin that made the version.
	ContextKeyPlugin = "version.plugin"
)

// getVersionContext is the context a command is offered over for the version.
func getVersionContext(version domain.TextVersion, isLive bool) pluginport.PluginContext {
	return pluginport.PluginContext{
		ContextKeyLive:   strconv.FormatBool(isLive),
		ContextKeyKind:   string(version.Kind),
		ContextKeyPlugin: version.PluginID,
	}
}

// listCandidates is the versions in whose context every condition of the command holds.
func listCandidates(command pluginport.PluginCommand, versions []domain.TextVersion, live domain.TextVersion, hasLive bool) []domain.TextVersion {
	var candidates []domain.TextVersion
	for _, version := range versions {
		isLive := hasLive && version.GetID() == live.GetID()
		if command.CanRunIn(getVersionContext(version, isLive)) {
			candidates = append(candidates, version)
		}
	}
	return candidates
}

// RunMenuEntry runs the command of an entry the file's menu offers now, over
// the entry's version. An entry the menu does not offer is ErrInvalidVersion.
func (u *FileMenu) RunMenuEntry(ctx context.Context, v domain.Vault, path string, entry MenuEntry) error {
	entries, err := u.GetMenu(ctx, v, path)
	if err != nil {
		return err
	}
	isInMenu := slices.ContainsFunc(entries, func(candidate MenuEntry) bool {
		return candidate.PluginID == entry.PluginID && candidate.Command == entry.Command && candidate.Version == entry.Version
	})
	if !isInMenu {
		return fmt.Errorf("%w: %s of %s is not offered over %s", ErrInvalidVersion, entry.Command, entry.PluginID, path)
	}
	call := pluginport.PluginCommandCall{PluginID: entry.PluginID, Command: entry.Command, Path: path}
	if entry.Version != "" {
		call.Arguments = map[string]string{pluginport.CommandArgumentVersion: string(entry.Version)}
	}
	_, err = u.commands.RunCommand(ctx, call)
	return err
}

// The errors of a menu entry the file's menu does not hold, for a driving adapter.
var (
	ErrNoText            = textport.ErrNoText
	ErrNotTextPlugin     = textport.ErrNotTextPlugin
	ErrInvalidVersion    = textport.ErrInvalidVersion
	ErrSourceNotAccepted = textport.ErrSourceNotAccepted
)
