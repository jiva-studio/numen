package command

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

func newRegistry(t *testing.T) *Registry {
	t.Helper()
	registry := New()
	err := registry.Add("greet", func(_ context.Context, request Request) (Result, error) {
		if request.Arguments.GetString("times") == "-1" {
			return Result{}, fmt.Errorf("%w: times cannot be negative", ErrBadArguments)
		}
		if request.Arguments.GetString("loud") == "true" {
			return Result{Message: "HELLO " + request.Arguments.GetString("name"), OpenedPath: request.Path}, nil
		}
		return Result{Message: "hello " + request.Arguments.GetString("name")}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Add("fail", func(context.Context, Request) (Result, error) {
		return Result{}, errors.New("the disk is full")
	}); err != nil {
		t.Fatal(err)
	}
	return registry
}

func run(t *testing.T, handler Handler, request *v1.RunCommandRequest) *v1.RunCommandResponse {
	t.Helper()
	res, err := service{handler: handler}.RunCommand(t.Context(), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	return res.Msg
}

func TestCommand_RunsWithFileAndArgumentsAndReturnsOutput(t *testing.T) {
	res := run(t, newRegistry(t), &v1.RunCommandRequest{
		Command: "greet", Path: "a.md", Arguments: map[string]string{"name": "Ann", "loud": "true"},
	})

	if res.GetError() != v1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED || res.GetMessage() != "HELLO Ann" || res.GetOpenedPath() != "a.md" {
		t.Errorf("the command answered %v", res)
	}
}

func TestCommand_FailureReturnsCodeAndMessage(t *testing.T) {
	cases := map[string]struct {
		request *v1.RunCommandRequest
		want    v1.CommandErrorCode
		message string
	}{
		"a command not added": {
			&v1.RunCommandRequest{Command: "shred"}, v1.CommandErrorCode_COMMAND_ERROR_CODE_UNKNOWN_COMMAND,
			"no command of that id: shred",
		},
		"arguments it cannot run with": {
			&v1.RunCommandRequest{Command: "greet", Arguments: map[string]string{"times": "-1"}},
			v1.CommandErrorCode_COMMAND_ERROR_CODE_BAD_ARGUMENTS,
			"the arguments do not fit the command: times cannot be negative",
		},
		"a failure": {
			&v1.RunCommandRequest{Command: "fail"}, v1.CommandErrorCode_COMMAND_ERROR_CODE_FAILED, "the disk is full",
		},
	}
	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			res := run(t, newRegistry(t), one.request)

			if res.GetError() != one.want || res.GetMessage() != one.message {
				t.Errorf("the command answered %v, want %v %q", res, one.want, one.message)
			}
		})
	}
}

func TestAddRejectsDuplicateID(t *testing.T) {
	registry := newRegistry(t)

	err := registry.Add("greet", func(context.Context, Request) (Result, error) { return Result{}, nil })

	if !errors.Is(err, ErrDuplicate) {
		t.Errorf("a second command under the id is %v, want ErrDuplicate", err)
	}
}

func TestMountAndCanceledContextAndUnlockAfterRun(t *testing.T) {
	registry := newRegistry(t)
	h, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "tok"}, func(s *server.Server) error {
		Mount(s, registry)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	client := numenv1connect.NewPluginCommandServiceClient(srv.Client(), srv.URL, hostcall.WithBearer("tok"))
	res, err := client.RunCommand(t.Context(), connect.NewRequest(&v1.RunCommandRequest{
		Command:   "greet",
		Arguments: map[string]string{"name": "Bob"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetMessage() != "hello Bob" {
		t.Fatalf("got %q, want %q", res.Msg.GetMessage(), "hello Bob")
	}

	if err := registry.Add("extra", func(context.Context, Request) (Result, error) { return Result{Message: "ok"}, nil }); err != nil {
		t.Fatal(err)
	}

	canceledCtx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = service{handler: registry}.RunCommand(canceledCtx, connect.NewRequest(&v1.RunCommandRequest{Command: "fail"}))
	if connect.CodeOf(err) != connect.CodeCanceled {
		t.Fatalf("got %v, want CodeCanceled", err)
	}
}

func TestCommand_PassesTheItemPickedToTheRun(t *testing.T) {
	registry := New()
	var got Request
	if err := registry.Add("download", func(_ context.Context, request Request) (Result, error) {
		got = request
		return Result{}, nil
	}); err != nil {
		t.Fatal(err)
	}

	run(t, registry, &v1.RunCommandRequest{Command: "download", Path: "talk.md", ItemId: "audio"})

	if got.ItemID != "audio" || got.Path != "talk.md" {
		t.Errorf("the run received %+v", got)
	}
}

func TestIsSharedCommandIsTrueForTheIdsOfCommandsJSONOnly(t *testing.T) {
	for _, one := range ListSharedCommands() {
		if !IsSharedCommand(one.ID) {
			t.Errorf("%s is not taken as shared", one.ID)
		}
	}
	if IsSharedCommand("app:version.delete") || IsSharedCommand("proofreading.create") {
		t.Error("a command that is not shared is taken as shared")
	}
}
