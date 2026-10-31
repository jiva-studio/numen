package command_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

func newRegistry(t *testing.T) *command.Registry {
	t.Helper()
	registry := command.New()
	err := registry.Add("greet", func(_ context.Context, request command.Request) (command.Result, error) {
		if request.Arguments.GetString("times") == "-1" {
			return command.Result{}, fmt.Errorf("%w: times cannot be negative", command.ErrBadArguments)
		}
		if request.Arguments.GetString("loud") == "true" {
			return command.Result{Message: "HELLO " + request.Arguments.GetString("name"), OpenedPath: request.Path}, nil
		}
		return command.Result{Message: "hello " + request.Arguments.GetString("name")}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Add("fail", func(context.Context, command.Request) (command.Result, error) {
		return command.Result{}, errors.New("the disk is full")
	}); err != nil {
		t.Fatal(err)
	}
	return registry
}

func run(t *testing.T, handler command.Handler, request *v1.RunCommandRequest) *v1.RunCommandResponse {
	t.Helper()
	res, err := command.Service{Handler: handler}.RunCommand(t.Context(), connect.NewRequest(request))
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

	err := registry.Add("greet", func(context.Context, command.Request) (command.Result, error) { return command.Result{}, nil })

	if !errors.Is(err, command.ErrDuplicate) {
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
	version, err := texts.PutVersion(ctx, host.TextPut{Path: "a.pdf", Key: "k", Kind: "recognition", SourceHash: domain.HashContent(file), File: "k.jsonl"})
	if err != nil {
		t.Fatal(err)
	}
	run := command.NewDeleteTextRun(texts, store, "org.example.ocr")

	if _, err := run(ctx, command.Request{Command: "delete", Path: "a.pdf"}); !errors.Is(err, command.ErrBadArguments) {
		t.Fatalf("a call with no version: got %v, want ErrBadArguments", err)
	}
	arguments := command.Arguments{command.ArgumentVersion: string(version.GetID())}
	if _, err := run(ctx, command.Request{Command: "delete", Path: "a.pdf", Arguments: arguments}); err != nil {
		t.Fatal(err)
	}
	listing, err := texts.ListVersions(ctx, "a.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Versions) != 0 {
		t.Fatalf("the versions after the deletion: %+v", listing.Versions)
	}
}
