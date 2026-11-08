package service

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
)

func TestThePluginRunsNeitherDeleteTextNorTheCommandOfBefore(t *testing.T) {
	s := newScene(t)
	commands, err := newCommands(s.run)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := commands.RunCommand(t.Context(), command.Request{Command: "app:version.delete", Path: scanPath}); !errors.Is(err, command.ErrUnknownCommand) {
		t.Errorf("Delete text, which the core runs, answered %v, want ErrUnknownCommand", err)
	}
	if _, err := commands.RunCommand(t.Context(), command.Request{Command: "proofreading.delete", Path: scanPath}); !errors.Is(err, command.ErrUnknownCommand) {
		t.Errorf("the command of before answered %v, want ErrUnknownCommand", err)
	}
}
