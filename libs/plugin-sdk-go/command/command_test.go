package command

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
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

func TestNewDeleteTextRunDeletesTheVersionTheHostNames(t *testing.T) {
	ctx := t.Context()
	file := []byte("%PDF")
	texts := hosttest.NewTexts("v1", "org.example.ocr")
	texts.File("a.pdf", file)
	store := texts.GetStore("org.example.ocr")
	if err := host.WriteText(ctx, store, "k.jsonl", []domain.TextSegment{{Text: "a"}}); err != nil {
		t.Fatal(err)
	}
	version, err := texts.PutVersion(ctx, host.TextPut{Path: "a.pdf", Key: "k", Kind: "recognition", InputHash: domain.HashContent(file), File: "k.jsonl"})
	if err != nil {
		t.Fatal(err)
	}
	run := NewDeleteTextRun(texts, store, "org.example.ocr")

	if _, err := run(ctx, Request{Command: "delete", Path: "a.pdf"}); !errors.Is(err, ErrBadArguments) {
		t.Fatalf("a call with no version: got %v, want ErrBadArguments", err)
	}
	if _, err := run(ctx, Request{Command: "delete", Path: "a.pdf", Arguments: Arguments{ArgumentVersion: "missing"}}); err == nil {
		t.Fatal("expected error when deleting a missing version")
	}
	arguments := Arguments{ArgumentVersion: string(version.GetID())}
	res, err := run(ctx, Request{Command: "delete", Path: "a.pdf", Arguments: arguments})
	if err != nil {
		t.Fatal(err)
	}
	if want := "deleted " + string(version.GetID()) + " of a.pdf"; res.Message != want {
		t.Fatalf("got message %q, want %q", res.Message, want)
	}
	listing, err := texts.ListVersions(ctx, "a.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Versions) != 0 {
		t.Fatalf("the versions after the deletion: %+v", listing.Versions)
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
