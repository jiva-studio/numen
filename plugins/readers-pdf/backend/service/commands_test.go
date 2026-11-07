package service

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
)

func TestTheHandlerOfCreateTextWritesTheTextLayerOfTheFile(t *testing.T) {
	texts, server, _ := newVault(t, "books/tiny.pdf", "tiny.pdf")
	commands, err := newCommands(server)
	if err != nil {
		t.Fatal(err)
	}

	result, err := commands.RunCommand(t.Context(), command.Request{Command: commandCreateText, Path: "books/tiny.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	server.Wait()

	if result.Message == "" {
		t.Error("the handler answered no message")
	}
	if text := readLive(t, texts, "books/tiny.pdf"); text.Version().Kind != textKind {
		t.Errorf("the live version is of kind %q, want %q", text.Version().Kind, textKind)
	}
}

func TestTheHandlerOfCreateTextNeedsAFile(t *testing.T) {
	_, server, _ := newVault(t, "books/tiny.pdf", "tiny.pdf")
	commands, err := newCommands(server)
	if err != nil {
		t.Fatal(err)
	}

	_, err = commands.RunCommand(t.Context(), command.Request{Command: commandCreateText})
	if !errors.Is(err, command.ErrBadArguments) {
		t.Errorf("a run over no file answered %v, want ErrBadArguments", err)
	}
}
