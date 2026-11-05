package command_test

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/command"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
)

// conditionalCommands declares a command the menu of a file offers over a version and a command over
// the window.
type conditionalCommands struct{ pluginCommands }

func (conditionalCommands) ListCommands() []plugincommand.Entry {
	return []plugincommand.Entry{
		{PluginID: "md.numen.tools.proofreading", Command: plugincommand.Command{
			ID: "proofreading.create", Title: "Proofread text",
			When: []pluginport.PluginContextCondition{{Key: "version.live", Is: []string{"true"}}},
		}},
		{PluginID: "md.numen.tools.updater", Command: plugincommand.Command{ID: "updater.check", Title: "Check for updates"}},
	}
}

func TestListCommands_OmitsACommandWithConditions(t *testing.T) {
	res, err := command.NewHandler(shownVault{}, plugincommand.NewCommands(conditionalCommands{}, fileTypes{}, nil)).ListCommands(
		t.Context(), connect.NewRequest(&v1.ListCommandsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	var ids []string
	for _, one := range res.Msg.GetCommands() {
		ids = append(ids, one.GetId())
	}
	if len(ids) != 1 || ids[0] != "updater.check" {
		t.Errorf("got %v, want only updater.check", ids)
	}
}
