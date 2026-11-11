package service

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/testsupport"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/usecase/texts"
)

const pluginID = "md.numen.readers.epub"

func newTextVault(t *testing.T, name string, raw []byte) (*hosttest.Texts, *texts.Writer) {
	t.Helper()
	files := hosttest.New("vault")
	files.File(name, string(raw))
	versions := hosttest.NewTexts("vault", pluginID)
	versions.File(name, raw)
	versions.SetStore(pluginID, files.Store())
	writer := texts.NewWriter(
		func(id domain.VaultID) texts.Vault { return files.With(id) },
		func(id domain.VaultID) host.TextVersions { return versions.With(id) },
		func(id domain.VaultID) host.StoreWriter { return files.Store().With(id) },
		hosttest.NewQueue(), func(string, error) {})
	t.Cleanup(writer.Close)
	return versions, writer
}

func TestTheHandlerOfCreateTextWritesTheTextOfTheBook(t *testing.T) {
	versions, writer := newTextVault(t, "book.epub", testsupport.CreateImageEpub(t))
	commands, err := newCommands(writer)
	if err != nil {
		t.Fatal(err)
	}

	result, err := commands.RunCommand(t.Context(), command.Request{Command: command.CreateText, Path: "book.epub"})
	if err != nil {
		t.Fatal(err)
	}
	writer.Wait()

	if result.Message == "" {
		t.Error("the handler answered no message")
	}
	listing, err := versions.ListVersions(t.Context(), "book.epub")
	if err != nil || len(listing.Versions) != 1 || listing.Versions[0].Kind != texts.Kind {
		t.Errorf("the versions are %+v (%v), want one of kind %q", listing.Versions, err, texts.Kind)
	}
}

func TestTheHandlerOfCreateTextNeedsABook(t *testing.T) {
	_, writer := newTextVault(t, "book.epub", testsupport.CreateImageEpub(t))
	commands, err := newCommands(writer)
	if err != nil {
		t.Fatal(err)
	}

	_, err = commands.RunCommand(t.Context(), command.Request{Command: command.CreateText})
	if !errors.Is(err, command.ErrBadArguments) {
		t.Errorf("a run over no file answered %v, want ErrBadArguments", err)
	}
}
