package command_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/command"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
)

// versionItems offers two versions of every file, the first live, and records each run.
type versionItems struct {
	ran     []string
	deleted []string
}

func (v *versionItems) ListVersionItems(_ context.Context, _ domain.Vault, path string, _ plugin.CommandEntry) ([]text.VersionItem, error) {
	if path == "gone.mp3" {
		return nil, errors.New("gone.mp3 is not in the vault")
	}
	return []text.VersionItem{
		{Version: "org.example.ears/ears", Kind: "transcript", IsLive: true},
		{Version: "org.example.ears/older", Kind: "transcript"},
	}, nil
}

func (v *versionItems) RunVersionItem(
	_ context.Context, _ domain.Vault, path string, entry plugin.CommandEntry, version domain.TextVersionID,
) (plugin.CommandResult, error) {
	if version != "org.example.ears/ears" {
		return plugin.CommandResult{}, text.ErrInvalidVersion
	}
	v.ran = append(v.ran, entry.PluginID+":"+entry.Command.ID+" "+path+" "+string(version))
	return plugin.CommandResult{Message: "Proofread"}, nil
}

func newItemsHandler(items *versionItems) *command.Handler {
	return newHandler(newScenarios(conditionalCommands{}, conditionalCommands{}, fileTypes{}, nil, nil).withVersions(items))
}

func TestListCommands_ListsACommandWithItemsAndMarksIt(t *testing.T) {
	res, err := newItemsHandler(&versionItems{}).ListCommands(t.Context(), connect.NewRequest(&v1.ListCommandsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]bool{}
	for _, one := range res.Msg.GetCommands() {
		got[one.GetId()] = one.GetHasItems()
	}
	if want := map[string]bool{"proofreading.create": true, "updater.check": false}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func listItemsOfOne(t *testing.T, handler *command.Handler, pluginID, commandID, path string) *v1.CommandItemList {
	t.Helper()
	res, err := handler.ListCommandItems(t.Context(), connect.NewRequest(&v1.ListCommandItemsRequest{
		Path: path, Commands: []*v1.PluginCommandId{{PluginId: pluginID, Command: commandID}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Msg.GetLists()) != 1 {
		t.Fatalf("got %d lists for one command", len(res.Msg.GetLists()))
	}
	return res.Msg.GetLists()[0]
}

func TestListCommandItems_ListsTheVersionsOfTheFileByKind(t *testing.T) {
	list := listItemsOfOne(t, newItemsHandler(&versionItems{}), "md.numen.tools.proofreading", "proofreading.create", "talks/a.mp3")

	var got [][2]string
	for _, one := range list.GetItems() {
		got = append(got, [2]string{one.GetId(), one.GetTitle()})
	}
	want := [][2]string{{"org.example.ears/ears", "transcript (current)"}, {"org.example.ears/older", "transcript"}}
	if !reflect.DeepEqual(got, want) || list.GetError() != v1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED {
		t.Errorf("got %v, %v, want %v", got, list.GetError(), want)
	}
}

func TestListCommandItems_AnswersNoneForACommandWithoutItems(t *testing.T) {
	list := listItemsOfOne(t, newItemsHandler(&versionItems{}), "md.numen.tools.updater", "updater.check", "talks/a.mp3")

	if len(list.GetItems()) != 0 || list.GetError() != v1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED {
		t.Errorf("got %v, %v", list.GetItems(), list.GetError())
	}
}

func TestListCommandItems_AnswersAnUnknownCommandAndAFailedListWithACode(t *testing.T) {
	for name, tc := range map[string]struct {
		command, path string
		want          v1.CommandErrorCode
	}{
		"unknown":     {"proofreading.gone", "", v1.CommandErrorCode_COMMAND_ERROR_CODE_UNKNOWN_COMMAND},
		"failed list": {"proofreading.create", "gone.mp3", v1.CommandErrorCode_COMMAND_ERROR_CODE_FAILED},
	} {
		list := listItemsOfOne(t, newItemsHandler(&versionItems{}), "md.numen.tools.proofreading", tc.command, tc.path)
		if list.GetError() != tc.want || list.GetMessage() == "" {
			t.Errorf("%s: got %v %q, want %v", name, list.GetError(), list.GetMessage(), tc.want)
		}
	}
}

func TestListCommandItems_AnswersOneListPerCommandInTheOrderOfTheRequest(t *testing.T) {
	res, err := newItemsHandler(&versionItems{}).ListCommandItems(t.Context(), connect.NewRequest(&v1.ListCommandItemsRequest{
		Path: "talks/a.mp3",
		Commands: []*v1.PluginCommandId{
			{PluginId: "md.numen.tools.updater", Command: "updater.check"},
			{PluginId: "md.numen.tools.proofreading", Command: "proofreading.gone"},
			{PluginId: "md.numen.tools.proofreading", Command: "proofreading.create"},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}

	type answer struct {
		pluginID, command string
		items             int
		error             v1.CommandErrorCode
	}
	var got []answer
	for _, one := range res.Msg.GetLists() {
		got = append(got, answer{one.GetPluginId(), one.GetCommand(), len(one.GetItems()), one.GetError()})
	}
	want := []answer{
		{"md.numen.tools.updater", "updater.check", 0, v1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED},
		{"md.numen.tools.proofreading", "proofreading.gone", 0, v1.CommandErrorCode_COMMAND_ERROR_CODE_UNKNOWN_COMMAND},
		{"md.numen.tools.proofreading", "proofreading.create", 2, v1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRunPluginCommand_RunsACommandWithItemsOverTheItemPicked(t *testing.T) {
	items := &versionItems{}
	res, err := newItemsHandler(items).RunPluginCommand(t.Context(), connect.NewRequest(&v1.RunPluginCommandRequest{
		PluginId: "md.numen.tools.proofreading", Command: "proofreading.create", Path: "talks/a.mp3", ItemId: "org.example.ears/ears",
	}))
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"md.numen.tools.proofreading:proofreading.create talks/a.mp3 org.example.ears/ears"}; !reflect.DeepEqual(items.ran, want) {
		t.Errorf("ran %v, want %v", items.ran, want)
	}
	if res.Msg.GetError() != v1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED || res.Msg.GetMessage() != "Proofread" {
		t.Errorf("got %v %q", res.Msg.GetError(), res.Msg.GetMessage())
	}
}

func TestRunPluginCommand_PassesTheItemPickedToACommandWithoutVersionItems(t *testing.T) {
	plugins := &runRecorded{}
	handler := newHandler(newScenarios(plugins, plugins, fileTypes{}, nil, nil).withVersions(&versionItems{}))

	res, err := handler.RunPluginCommand(t.Context(), connect.NewRequest(&v1.RunPluginCommandRequest{
		PluginId: "md.numen.tools.updater", Command: "updater.check", ItemId: "beta",
	}))
	if err != nil {
		t.Fatal(err)
	}

	if res.Msg.GetError() != v1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED || len(plugins.ran) != 1 || plugins.ran[0].ItemID != "beta" {
		t.Errorf("got %v, ran %+v", res.Msg.GetError(), plugins.ran)
	}
}

// runRecorded is conditionalCommands recording the runs that reach the plugins.
type runRecorded struct {
	conditionalCommands
	ran []plugin.CommandCall
}

func (p *runRecorded) RunCommand(_ context.Context, call plugin.CommandCall) (plugin.CommandResult, error) {
	p.ran = append(p.ran, call)
	return plugin.CommandResult{}, nil
}

func TestRunPluginCommand_RejectsAnItemNotListedAndARunWithoutAnItem(t *testing.T) {
	for name, request := range map[string]*v1.RunPluginCommandRequest{
		"an item not offered": {PluginId: "md.numen.tools.proofreading", Command: "proofreading.create", Path: "talks/a.mp3", ItemId: "org.example.ears/older"},
		"no item":             {PluginId: "md.numen.tools.proofreading", Command: "proofreading.create", Path: "talks/a.mp3"},
	} {
		items := &versionItems{}
		res, err := newItemsHandler(items).RunPluginCommand(t.Context(), connect.NewRequest(request))
		if err != nil {
			t.Fatal(err)
		}
		if res.Msg.GetError() != v1.CommandErrorCode_COMMAND_ERROR_CODE_BAD_ARGUMENTS || len(items.ran) != 0 {
			t.Errorf("%s: got %v, ran %v", name, res.Msg.GetError(), items.ran)
		}
	}
}

// conditionalCommands declares a command with items, one per version of a file, and a command over
// the window.
type conditionalCommands struct{ pluginCommands }

func (conditionalCommands) ListCommands() []plugin.CommandEntry {
	return []plugin.CommandEntry{
		{PluginID: "md.numen.tools.proofreading", Command: manifest.Command{
			ID: "proofreading.create", Title: "Proofread text",
			When: "version.isLive",
		}},
		{PluginID: "md.numen.tools.updater", Command: manifest.Command{ID: "updater.check", Title: "Check for updates"}},
	}
}

// comparingCommands declares two commands with items and one without.
type comparingCommands struct{ pluginCommands }

func (comparingCommands) ListCommands() []plugin.CommandEntry {
	return []plugin.CommandEntry{
		{PluginID: "md.numen.tools.proofreading", Command: manifest.Command{ID: "proofreading.create", Title: "Proofread text", When: "version.isLive"}},
		{PluginID: "md.numen.tools.proofreading", Command: manifest.Command{ID: "proofreading.compare", Title: "Compare text", When: "version.isLive"}},
		{PluginID: "md.numen.tools.updater", Command: manifest.Command{ID: "updater.check", Title: "Check for updates"}},
	}
}

// brokenComparison lists the versions of versionItems, and fails for proofreading.compare.
type brokenComparison struct{ versionItems }

func (b *brokenComparison) ListVersionItems(ctx context.Context, v domain.Vault, path string, entry plugin.CommandEntry) ([]text.VersionItem, error) {
	if entry.Command.ID == "proofreading.compare" {
		return nil, errors.New("the comparison cannot read the versions")
	}
	return b.versionItems.ListVersionItems(ctx, v, path, entry)
}

func TestListCommandItems_AnswersTheOtherListsWhereOneCommandFails(t *testing.T) {
	handler := newHandler(newScenarios(comparingCommands{}, comparingCommands{}, fileTypes{}, nil, nil).withVersions(&brokenComparison{}))

	res, err := handler.ListCommandItems(t.Context(), connect.NewRequest(&v1.ListCommandItemsRequest{
		Path: "talks/a.mp3",
		Commands: []*v1.PluginCommandId{
			{PluginId: "md.numen.tools.proofreading", Command: "proofreading.create"},
			{PluginId: "md.numen.tools.proofreading", Command: "proofreading.compare"},
			{PluginId: "md.numen.tools.updater", Command: "updater.check"},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}

	lists := res.Msg.GetLists()
	if len(lists) != 3 || len(lists[0].GetItems()) != 2 || lists[0].GetError() != v1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED ||
		lists[1].GetError() != v1.CommandErrorCode_COMMAND_ERROR_CODE_FAILED || lists[1].GetMessage() == "" ||
		lists[2].GetError() != v1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED {
		t.Errorf("lists %v, want the failure of the comparison alone", lists)
	}
}
