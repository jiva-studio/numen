package service

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
)

func TestTheHandlerOfCreateTextWritesTheTextOfTheBook(t *testing.T) {
	texts, server := newTextVault(t, "book.epub", createImageEpub(t))
	commands, err := newCommands(server)
	if err != nil {
		t.Fatal(err)
	}

	result, err := commands.RunCommand(t.Context(), command.Request{Command: command.CreateText, Path: "book.epub"})
	if err != nil {
		t.Fatal(err)
	}
	server.Wait()

	if result.Message == "" {
		t.Error("the handler answered no message")
	}
	if text := readLive(t, texts, "book.epub"); text.Version().Kind != textKind {
		t.Errorf("the live version is of kind %q, want %q", text.Version().Kind, textKind)
	}
}

func TestTheHandlerOfCreateTextNeedsABook(t *testing.T) {
	_, server := newTextVault(t, "book.epub", createImageEpub(t))
	commands, err := newCommands(server)
	if err != nil {
		t.Fatal(err)
	}

	_, err = commands.RunCommand(t.Context(), command.Request{Command: command.CreateText})
	if !errors.Is(err, command.ErrBadArguments) {
		t.Errorf("a run over no file answered %v, want ErrBadArguments", err)
	}
}
