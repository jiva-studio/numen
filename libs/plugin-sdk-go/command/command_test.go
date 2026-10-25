package command_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

func newRegistry(t *testing.T) *command.Registry {
	t.Helper()
	registry := command.New()
	err := registry.Add("greet", func(_ context.Context, request command.Request) (string, error) {
		times, _ := request.Arguments.GetNumber("times")
		if times < 0 {
			return "", fmt.Errorf("%w: times cannot be negative", command.ErrBadArguments)
		}
		if request.Arguments.GetBool("loud") {
			return "HELLO " + request.Arguments.GetString("name") + " in " + request.Path, nil
		}
		return "hello " + request.Arguments.GetString("name"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Add("fail", func(context.Context, command.Request) (string, error) {
		return "", errors.New("the disk is full")
	}); err != nil {
		t.Fatal(err)
	}
	return registry
}

func run(t *testing.T, handler command.Handler, request *numenv1.RunCommandRequest) *numenv1.RunCommandResponse {
	t.Helper()
	res, err := command.Service{Handler: handler}.RunCommand(t.Context(), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	return res.Msg
}

func TestACommandRunsWithItsFileAndArgumentsAndAnswersWhatItSaid(t *testing.T) {
	res := run(t, newRegistry(t), &numenv1.RunCommandRequest{
		Command: "greet", Path: "a.md", Arguments: map[string]string{"name": "Ann", "loud": "true"},
	})

	if res.GetError() != numenv1.CommandErrorCode_COMMAND_ERROR_CODE_UNSPECIFIED || res.GetMessage() != "HELLO Ann in a.md" {
		t.Errorf("the command answered %v", res)
	}
}

func TestACommandThatDidNotSucceedIsAnsweredWithItsCodeAndItsWords(t *testing.T) {
	cases := map[string]struct {
		request *numenv1.RunCommandRequest
		want    numenv1.CommandErrorCode
		message string
	}{
		"a command not added": {
			&numenv1.RunCommandRequest{Command: "shred"}, numenv1.CommandErrorCode_COMMAND_ERROR_CODE_UNKNOWN_COMMAND,
			"no command of that id: shred",
		},
		"arguments it cannot run with": {
			&numenv1.RunCommandRequest{Command: "greet", Arguments: map[string]string{"times": "-1"}},
			numenv1.CommandErrorCode_COMMAND_ERROR_CODE_BAD_ARGUMENTS,
			"the arguments do not fit the command: times cannot be negative",
		},
		"a failure": {
			&numenv1.RunCommandRequest{Command: "fail"}, numenv1.CommandErrorCode_COMMAND_ERROR_CODE_FAILED, "the disk is full",
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

func TestACommandIsAddedUnderAnIDOnce(t *testing.T) {
	registry := newRegistry(t)

	err := registry.Add("greet", func(context.Context, command.Request) (string, error) { return "", nil })

	if !errors.Is(err, command.ErrDuplicate) {
		t.Errorf("a second command under the id is %v, want ErrDuplicate", err)
	}
}

func TestAnArgumentLeftOutIsAbsent(t *testing.T) {
	arguments := command.Arguments{"name": "Ann"}

	if !arguments.Has("name") || arguments.Has("times") || arguments.GetBool("loud") || arguments.GetString("label") != "" {
		t.Error("an argument left out is read as given")
	}
	if _, isGiven := arguments.GetNumber("times"); isGiven {
		t.Error("a number left out is read as given")
	}
}
