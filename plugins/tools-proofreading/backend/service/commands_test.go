package service

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
)

func TestTheHandlerOfDeleteTextTakesTheVersionTheHostNames(t *testing.T) {
	s := newScene(t)
	commands, err := newCommands(s.run, Host{Store: s.store, Texts: s.texts, Queue: s.queue})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := commands.RunCommand(t.Context(), command.Request{Command: "app:version.delete", Path: scanPath}); !errors.Is(err, command.ErrBadArguments) {
		t.Errorf("a deletion naming no version answered %v, want ErrBadArguments", err)
	}
	if _, err := commands.RunCommand(t.Context(), command.Request{Command: "proofreading.delete", Path: scanPath}); !errors.Is(err, command.ErrUnknownCommand) {
		t.Errorf("the command of before answered %v, want ErrUnknownCommand", err)
	}
}
