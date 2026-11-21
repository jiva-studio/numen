package service_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
)

func runCreate(t *testing.T, f *fixture, folder, title, query string) (command.Result, error) {
	t.Helper()
	commands, err := f.app.NewCommands()
	if err != nil {
		t.Fatal(err)
	}
	return commands.RunCommand(t.Context(), command.Request{
		Command: "search.create", Path: folder, Arguments: command.Arguments{"title": title, "query": query},
	})
}

func TestCreateCommandWritesTheNoteInTheFolderOfTheRow(t *testing.T) {
	f := newFixture(t, 10)

	result, err := runCreate(t, f, "Searches", "Winter storms", stormsQuery)
	if err != nil {
		t.Fatal(err)
	}

	if result.OpenedPath != stormsPath {
		t.Errorf("OpenedPath = %q, want %q", result.OpenedPath, stormsPath)
	}
	content, _ := readNote(t, f, stormsPath)
	if !strings.Contains(content, stormsQuery) {
		t.Errorf("note = %q, want the query", content)
	}
}

func TestCreateCommandUsesTheDefaultFolderWithoutARow(t *testing.T) {
	f := newFixture(t, 10)

	result, err := runCreate(t, f, "", "Winter storms", wordsQuery)
	if err != nil {
		t.Fatal(err)
	}

	if result.OpenedPath != "Winter storms.md" {
		t.Errorf("OpenedPath = %q, want Winter storms.md", result.OpenedPath)
	}
}

func TestCreateCommandRefusesAQueryTheCoreRejects(t *testing.T) {
	f := newFixture(t, 10)

	_, err := runCreate(t, f, "Searches", "Broken", brokenQuery)

	if err == nil || !strings.Contains(err.Error(), "Syntax error") {
		t.Fatalf("err = %v, want the problem of the query", err)
	}
	if _, isThere := f.vault.Content("Searches/Broken.md"); isThere {
		t.Error("a note was written for an invalid query")
	}
}
