package command_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/command"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
)

// versionItems offers two versions of every file, the first live, and records each run.
type versionItems struct {
	ran []string
}

func (v *versionItems) ListVersionItems(_ context.Context, _ domain.Vault, path string, _ pluginport.PluginCommandEntry) ([]textversion.VersionItem, error) {
	if path == "gone.mp3" {
		return nil, errors.New("gone.mp3 is not in the vault")
	}
	return []textversion.VersionItem{
		{Version: "org.example.ears/ears", Kind: "transcript", IsLive: true},
		{Version: "org.example.ears/older", Kind: "transcript"},
	}, nil
}

func (v *versionItems) RunVersionItem(
	_ context.Context, _ domain.Vault, path string, entry pluginport.PluginCommandEntry, version domain.TextVersionID,
) (pluginport.PluginCommandResult, error) {
	if version != "org.example.ears/ears" {
		return pluginport.PluginCommandResult{}, textversion.ErrInvalidVersion
	}
	v.ran = append(v.ran, entry.PluginID+":"+entry.Command.ID+" "+path+" "+string(version))
	return pluginport.PluginCommandResult{Message: "Proofread"}, nil
}

func newItemsHandler(items *versionItems) *command.Handler {
	return command.NewHandler(shownVault{}, plugincommand.NewCommands(conditionalCommands{}, fileTypes{}, nil), items)
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

func TestListCommandItems_ListsTheVersionsOfTheFileByKind(t *testing.T) {
	res, err := newItemsHandler(&versionItems{}).ListCommandItems(t.Context(), connect.NewRequest(&v1.ListCommandItemsRequest{
		PluginId: "md.numen.tools.proofreading", Command: "proofreading.create", Path: "talks/a.mp3",
	}))
	if err != nil {
		t.Fatal(err)
	}

	var got [][2]string
	for _, one := range res.Msg.GetItems() {
		got = append(got, [2]string{one.GetId(), one.GetTitle()})
	}
	want := [][2]string{{"org.example.ears/ears", "transcript (live)"}, {"org.example.ears/older", "transcript"}}
	if !reflect.DeepEqual(got, want) || res.Msg.GetError() != v1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED {
		t.Errorf("got %v, %v, want %v", got, res.Msg.GetError(), want)
	}
}

func TestListCommandItems_AnswersNoneForACommandWithoutItems(t *testing.T) {
	res, err := newItemsHandler(&versionItems{}).ListCommandItems(t.Context(), connect.NewRequest(&v1.ListCommandItemsRequest{
		PluginId: "md.numen.tools.updater", Command: "updater.check", Path: "talks/a.mp3",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Msg.GetItems()) != 0 || res.Msg.GetError() != v1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED {
		t.Errorf("got %v, %v", res.Msg.GetItems(), res.Msg.GetError())
	}
}

func TestListCommandItems_AnswersAnUnknownCommandAndAFailedListWithACode(t *testing.T) {
	for name, tc := range map[string]struct {
		request *v1.ListCommandItemsRequest
		want    v1.CommandErrorCode
	}{
		"unknown":     {&v1.ListCommandItemsRequest{PluginId: "md.numen.tools.proofreading", Command: "proofreading.gone"}, v1.CommandErrorCode_COMMAND_ERROR_CODE_UNKNOWN_COMMAND},
		"failed list": {&v1.ListCommandItemsRequest{PluginId: "md.numen.tools.proofreading", Command: "proofreading.create", Path: "gone.mp3"}, v1.CommandErrorCode_COMMAND_ERROR_CODE_FAILED},
	} {
		res, err := newItemsHandler(&versionItems{}).ListCommandItems(t.Context(), connect.NewRequest(tc.request))
		if err != nil {
			t.Fatal(err)
		}
		if res.Msg.GetError() != tc.want || res.Msg.GetMessage() == "" {
			t.Errorf("%s: got %v %q, want %v", name, res.Msg.GetError(), res.Msg.GetMessage(), tc.want)
		}
	}
}

func TestRunPluginCommand_RunsACommandWithItemsOverTheItemPicked(t *testing.T) {
	items := &versionItems{}
	res, err := newItemsHandler(items).RunPluginCommand(t.Context(), connect.NewRequest(&v1.RunPluginCommandRequest{
		PluginId: "md.numen.tools.proofreading", Command: "proofreading.create", Path: "talks/a.mp3", Argument: "org.example.ears/ears",
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

func TestRunPluginCommand_RejectsAnItemNotListedAndAnArgumentToACommandWithoutItems(t *testing.T) {
	for name, request := range map[string]*v1.RunPluginCommandRequest{
		"an item not offered":     {PluginId: "md.numen.tools.proofreading", Command: "proofreading.create", Path: "talks/a.mp3", Argument: "org.example.ears/older"},
		"no item":                 {PluginId: "md.numen.tools.proofreading", Command: "proofreading.create", Path: "talks/a.mp3"},
		"a command without items": {PluginId: "md.numen.tools.updater", Command: "updater.check", Argument: "org.example.ears/ears"},
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

func (conditionalCommands) ListCommands() []plugincommand.Entry {
	return []plugincommand.Entry{
		{PluginID: "md.numen.tools.proofreading", Command: plugincommand.Command{
			ID: "proofreading.create", Title: "Proofread text",
			When: "version.isLive",
		}},
		{PluginID: "md.numen.tools.updater", Command: plugincommand.Command{ID: "updater.check", Title: "Check for updates"}},
	}
}
