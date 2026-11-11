package service

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/internal/testsupport"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/usecase/extract"
)

func TestTheHandlerOfCreateTextWritesTheTextLayerOfTheFile(t *testing.T) {
	texts, server, _ := testsupport.NewVault(t, "books/tiny.pdf", "tiny.pdf")
	commands, err := newCommands(server)
	if err != nil {
		t.Fatal(err)
	}

	result, err := commands.RunCommand(t.Context(), command.Request{Command: command.CreateText, Path: "books/tiny.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	server.Wait()

	if result.Message == "" {
		t.Error("the handler answered no message")
	}
	if text := testsupport.ReadLive(t, texts, "books/tiny.pdf"); text.Version().Kind != extract.TextKind {
		t.Errorf("the live version is of kind %q, want %q", text.Version().Kind, extract.TextKind)
	}
}

func TestTheHandlerOfCreateTextNeedsAFile(t *testing.T) {
	_, server, _ := testsupport.NewVault(t, "books/tiny.pdf", "tiny.pdf")
	commands, err := newCommands(server)
	if err != nil {
		t.Fatal(err)
	}

	_, err = commands.RunCommand(t.Context(), command.Request{Command: command.CreateText})
	if !errors.Is(err, command.ErrBadArguments) {
		t.Errorf("a run over no file answered %v, want ErrBadArguments", err)
	}
}
