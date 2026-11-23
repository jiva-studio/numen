package service_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
)

func runCreate(t *testing.T, f *fixture, folder, title string) (command.Result, error) {
	t.Helper()
	commands, err := f.app.NewCommands()
	if err != nil {
		t.Fatal(err)
	}
	return commands.RunCommand(t.Context(), command.Request{
		Command: "search.create", Path: folder, Arguments: command.Arguments{"title": title},
	})
}

func TestCreateCommandWritesTheNoteInTheFolderOfTheRow(t *testing.T) {
	f := newFixture(t, 10)

	result, err := runCreate(t, f, "Searches", "Winter storms")
	if err != nil {
		t.Fatal(err)
	}

	if result.OpenedPath != stormsPath {
		t.Errorf("OpenedPath = %q, want %q", result.OpenedPath, stormsPath)
	}
	if _, isThere := f.vault.ReadContent(stormsPath); !isThere {
		t.Error("no note was written")
	}
}

func TestCreateCommandUsesTheDefaultFolderWithoutARow(t *testing.T) {
	f := newFixture(t, 10)

	result, err := runCreate(t, f, "", "Winter storms")
	if err != nil {
		t.Fatal(err)
	}

	if result.OpenedPath != "Winter storms.md" {
		t.Errorf("OpenedPath = %q, want Winter storms.md", result.OpenedPath)
	}
}

func TestCreateCommandRefusesAnEmptyTitle(t *testing.T) {
	f := newFixture(t, 10)

	_, err := runCreate(t, f, "Searches", "  ")

	if err == nil || !strings.Contains(err.Error(), "title") {
		t.Fatalf("err = %v, want the empty title", err)
	}
}
