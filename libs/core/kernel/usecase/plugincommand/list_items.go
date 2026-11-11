package plugincommand

import (
	"context"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
)

// CommandID names a command of a plugin; the application commands have no plugin.
type CommandID struct {
	PluginID string
	Command  string
}

// Item is one thing a command with items runs over.
type Item struct {
	ID     string
	Title  string
	Detail string
}

// ItemList is the items one command offers, or why they cannot be listed.
type ItemList struct {
	CommandID
	Items []Item
	Err   error
}

// ListItems lists the items each command offers over a file.
type ListItems struct {
	list     List
	versions VersionItems
}

// NewListItems makes the scenario that lists the items of the commands of the list. versions are the text
// versions that a command with items lists; without them such a command lists none.
func NewListItems(list List, versions VersionItems) ListItems {
	return ListItems{list: list, versions: versions}
}

// Execute lists the items each command offers over the path, one list per command in the order given:
// the text versions of the file for a command with items, none for any other command. A command that is
// unknown, or whose items cannot be listed, has a list with the error.
func (u ListItems) Execute(ctx context.Context, v domain.Vault, path string, commands []CommandID) []ItemList {
	lists := make([]ItemList, 0, len(commands))
	for _, one := range commands {
		list := u.listOne(ctx, v, path, one)
		list.CommandID = one
		lists = append(lists, list)
	}
	return lists
}

func (u ListItems) listOne(ctx context.Context, v domain.Vault, path string, id CommandID) ItemList {
	if isDeleteText(id.PluginID, id.Command) {
		return u.newList(u.listEveryVersion(ctx, v, path))
	}
	entry, found := findEntry(u.list, id.PluginID, id.Command)
	if !found {
		return ItemList{Err: fmt.Errorf("%w: %s of %s", ErrUnknownCommand, id.Command, id.PluginID)}
	}
	if !hasItems(u.list.conditions, entry.Command) {
		return ItemList{}
	}
	if u.versions == nil {
		return ItemList{Err: errNoVersions}
	}
	return u.newList(u.versions.ListVersionItems(ctx, v, path, entry))
}

func (u ListItems) listEveryVersion(ctx context.Context, v domain.Vault, path string) ([]text.VersionItem, error) {
	if u.versions == nil {
		return nil, errNoVersions
	}
	return u.versions.ListEveryVersion(ctx, v, path)
}

func (u ListItems) newList(versions []text.VersionItem, err error) ItemList {
	if err != nil {
		return ItemList{Err: err}
	}
	items := make([]Item, 0, len(versions))
	for _, one := range versions {
		parts := u.list.commands.GetParts(one.PluginID)
		items = append(items, Item{ID: string(one.Version), Title: getVersionTitle(one, parts), Detail: getVersionDetail(one, parts)})
	}
	return ItemList{Items: items}
}

func getVersionTitle(item text.VersionItem, parts plugin.Parts) string {
	title := item.Kind
	if declared, ok := parts.TextKinds[item.Kind]; ok {
		title = declared
	}
	if title == "" {
		title = "text"
	}
	if item.IsLive {
		return title + " (current)"
	}
	return title
}

func getVersionDetail(item text.VersionItem, parts plugin.Parts) string {
	maker := parts.Name
	if maker == "" {
		maker = item.PluginID
	}
	if item.CreateTime.IsZero() {
		return maker
	}
	if maker == "" {
		return item.CreateTime.Format(time.DateOnly)
	}
	return maker + " · " + item.CreateTime.Format(time.DateOnly)
}
