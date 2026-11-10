package container_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
)

// The size a note is cut at is a setting of the embedding plugin, and the core settings hold none.

func TestTheSettingsFileHasNoIndexingSection(t *testing.T) {
	if _, has := reflect.TypeFor[settings.File]().FieldByName("Indexing"); has {
		t.Error("the settings still have an Indexing section")
	}
	raw, err := json.Marshal(container.DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"indexing"`) {
		t.Errorf("the default settings name an indexing section:\n%s", raw)
	}
}

func TestAFirstRunWritesNoIndexingSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "numen", "numen.json")
	if _, err := openSettingsAt(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"indexing"`) || strings.Contains(string(raw), `"max_tokens"`) {
		t.Errorf("the file written names the cut of a note:\n%s", raw)
	}
}

func TestAFileWithTheOldIndexingKeyLoadsAndSetsNoChunkSize(t *testing.T) {
	path := writeSettingsFile(t, `{"indexing":{"embedding":{"max_tokens":128}}}`)
	cfg, err := openSettingsAt(path)
	if err != nil {
		t.Fatalf("a file with the old key was refused: %v", err)
	}

	chunker := container.NewConfig(container.Config{
		Paths: container.Paths{SettingsPath: path},
	}).SetSettings(cfg).CutNotes()
	if chunker.Sizes.Limit != chunking.DefaultLimit {
		t.Errorf("notes are cut at %d characters, want the default %d: the old key must name no cut", chunker.Sizes.Limit, chunking.DefaultLimit)
	}
}

func TestDefaultSettingsChunkNotesAtTheDefaultLimit(t *testing.T) {
	cfg := container.NewConfig(container.Config{}).SetSettings(container.DefaultSettings())

	if got := cfg.CutNotes().Sizes.Limit; got != chunking.DefaultLimit {
		t.Errorf("notes are cut at %d characters, want the default %d", got, chunking.DefaultLimit)
	}
}
