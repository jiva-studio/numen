package command_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/command"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/commandbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/manifestlist"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
)

// repositoryPlugins is the plugins folder of the repository.
const repositoryPlugins = "../../../../../../../plugins"

func newRepositoryHandler(b *testing.B) *command.Handler {
	b.Helper()
	host := plugin.NewHost(testsupport.GetMachine(), "desktop.editor", repositoryPlugins)
	b.Cleanup(func() {
		if err := host.TeardownAll(context.WithoutCancel(b.Context()), 0); err != nil {
			b.Error(err)
		}
	})
	if err := host.Discover(); err != nil {
		b.Fatal(err)
	}
	commands := plugincommand.NewCommands(struct {
		manifestlist.Manifests
		commandbridge.Commands
	}{manifestlist.Open(host), commandbridge.Open(host)}, host, nil, nil)
	return command.NewHandler(shownVault{}, commands, &versionItems{})
}

func BenchmarkListCommandItems_OfASharedCommandOverTheRepositoryPlugins(b *testing.B) {
	handler := newRepositoryHandler(b)
	request := connect.NewRequest(&v1.ListCommandItemsRequest{
		Path: "scans/page.png", Commands: []*v1.PluginCommandId{{PluginId: "md.numen.engines.ocr", Command: "app:file.createText"}},
	})
	for b.Loop() {
		res, err := handler.ListCommandItems(b.Context(), request)
		if err != nil || res.Msg.GetLists()[0].GetError() != v1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED {
			b.Fatalf("got %v, %v", res, err)
		}
	}
}

func BenchmarkListCommands_OverTheRepositoryPlugins(b *testing.B) {
	handler := newRepositoryHandler(b)
	request := connect.NewRequest(&v1.ListCommandsRequest{})
	for b.Loop() {
		if _, err := handler.ListCommands(b.Context(), request); err != nil {
			b.Fatal(err)
		}
	}
}
