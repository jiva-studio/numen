package plugincommand_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
)

// proofreadingVersions offers two versions of every file and records what is run and deleted.
type proofreadingVersions struct {
	ran     []string
	deleted []string
}

func (p *proofreadingVersions) ListVersionItems(context.Context, domain.Vault, string, plugin.CommandEntry) ([]text.VersionItem, error) {
	return []text.VersionItem{
		{Version: "org.example.ears/ears", Kind: "transcript", IsLive: true, PluginID: "org.example.ears", CreateTime: time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)},
		{Version: "org.example.ears/older", Kind: "transcript", PluginID: "org.example.ears"},
	}, nil
}

func (p *proofreadingVersions) RunVersionItem(
	_ context.Context, _ domain.Vault, path string, entry plugin.CommandEntry, version domain.TextVersionID,
) (plugin.CommandResult, error) {
	if version != "org.example.ears/ears" {
		return plugin.CommandResult{}, text.ErrInvalidVersion
	}
	p.ran = append(p.ran, entry.Command.ID+" "+path+" "+string(version))
	return plugin.CommandResult{Message: "Proofread"}, nil
}

func (p *proofreadingVersions) ListEveryVersion(ctx context.Context, v domain.Vault, path string) ([]text.VersionItem, error) {
	return p.ListVersionItems(ctx, v, path, plugin.CommandEntry{})
}

func (p *proofreadingVersions) DeleteVersion(_ context.Context, _ domain.Vault, path string, version domain.TextVersionID) error {
	if version != "org.example.ears/older" {
		return text.ErrInvalidVersion
	}
	p.deleted = append(p.deleted, path+" "+string(version))
	return nil
}

func newProofreading() *declaredCommands {
	return &declaredCommands{entries: []plugin.CommandEntry{
		{PluginID: "org.example.ears", Command: manifest.Command{ID: "proofread", Title: "Proofread", When: "version.kind == 'transcript'"}},
		{PluginID: "org.example.ears", Command: manifest.Command{ID: "tidy", Title: "Tidy"}},
	}}
}

func TestRunCommandRunsACommandWithItemsOverTheVersionItsItemNames(t *testing.T) {
	declared, versions := newProofreading(), &proofreadingVersions{}
	commands := newScenarios(declared, declared, getTypes(), nil, nil).withVersions(versions)

	result, err := commands.RunCommand.Execute(t.Context(), getVault(), plugin.CommandCall{
		PluginID: "org.example.ears", Command: "proofread", Path: "talks/a.mp3", ItemID: "org.example.ears/ears",
	})

	if err != nil || result.Message != "Proofread" || len(versions.ran) != 1 || len(declared.calls) != 0 {
		t.Errorf("the run answered %+v (%v), ran %v, and the plugin was called %v times", result, err, versions.ran, len(declared.calls))
	}
}

func TestRunCommandRunsACommandWithoutItemsInItsPlugin(t *testing.T) {
	declared := newProofreading()
	commands := newScenarios(declared, declared, getTypes(), nil, nil).withVersions(&proofreadingVersions{})

	_, err := commands.RunCommand.Execute(t.Context(), getVault(), plugin.CommandCall{PluginID: "org.example.ears", Command: "tidy"})

	if err != nil || len(declared.calls) != 1 {
		t.Errorf("the command answered %v and the plugin was called %d times", err, len(declared.calls))
	}
}

func TestRunCommandRefusesACommandWithItemsOverAnItemItDoesNotOffer(t *testing.T) {
	declared := newProofreading()
	commands := newScenarios(declared, declared, getTypes(), nil, nil).withVersions(&proofreadingVersions{})

	for name, call := range map[string]plugin.CommandCall{
		"no item":         {PluginID: "org.example.ears", Command: "proofread", Path: "talks/a.mp3"},
		"a version none":  {PluginID: "org.example.ears", Command: "proofread", Path: "talks/a.mp3", ItemID: "org.example.ears/gone"},
		"delete no item":  {Command: text.DeleteCommand, Path: "talks/a.mp3"},
		"delete a bad id": {Command: text.DeleteCommand, Path: "talks/a.mp3", ItemID: "org.example.ears/gone"},
	} {
		if _, err := commands.RunCommand.Execute(t.Context(), getVault(), call); !errors.Is(err, plugincommand.ErrBadArguments) {
			t.Errorf("%s: the call answered %v, want bad arguments", name, err)
		}
	}
}

func TestRunCommandDeletesTheVersionItsItemNames(t *testing.T) {
	declared, versions := newProofreading(), &proofreadingVersions{}
	commands := newScenarios(declared, declared, getTypes(), nil, nil).withVersions(versions)

	_, err := commands.RunCommand.Execute(t.Context(), getVault(), plugin.CommandCall{
		Command: text.DeleteCommand, Path: "talks/a.mp3", ItemID: "org.example.ears/older",
	})

	if err != nil || len(versions.deleted) != 1 {
		t.Errorf("the delete answered %v and deleted %v", err, versions.deleted)
	}
}

func TestRunCommandWithoutVersionsRefusesACommandWithItems(t *testing.T) {
	declared := newProofreading()
	commands := newScenarios(declared, declared, getTypes(), nil, nil)

	_, err := commands.RunCommand.Execute(t.Context(), getVault(), plugin.CommandCall{
		PluginID: "org.example.ears", Command: "proofread", ItemID: "org.example.ears/ears",
	})

	if err == nil || errors.Is(err, plugincommand.ErrBadArguments) {
		t.Errorf("a window with no text versions ran a command with items: %v", err)
	}
}

func TestListItemsListsTheVersionsOfACommandWithItemsAndNothingForOthers(t *testing.T) {
	declared := newProofreading()
	commands := newScenarios(declared, declared, getTypes(), nil, nil).withVersions(&proofreadingVersions{})

	lists := commands.ListItems.Execute(t.Context(), getVault(), "talks/a.mp3", []plugincommand.CommandID{
		{PluginID: "org.example.ears", Command: "proofread"},
		{PluginID: "org.example.ears", Command: "tidy"},
		{PluginID: "org.example.ears", Command: "missing"},
		{Command: text.DeleteCommand},
	})

	if len(lists) != 4 {
		t.Fatalf("%d lists for four commands", len(lists))
	}
	if got := lists[0].Items; len(got) != 2 || got[0].Title != "transcript (current)" || got[0].Detail != "org.example.ears · 2026-03-04" {
		t.Errorf("the items of the command with items are %+v", got)
	}
	if lists[1].Err != nil || len(lists[1].Items) != 0 {
		t.Errorf("a command without items listed %+v (%v)", lists[1].Items, lists[1].Err)
	}
	if !errors.Is(lists[2].Err, plugincommand.ErrUnknownCommand) {
		t.Errorf("an unknown command listed %v", lists[2].Err)
	}
	if lists[3].Err != nil || len(lists[3].Items) != 2 {
		t.Errorf("Delete text listed %+v (%v)", lists[3].Items, lists[3].Err)
	}
}
