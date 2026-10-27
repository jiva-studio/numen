package textfiles_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/textfiles"
	"github.com/jiva-studio/numen/libs/core/port/textport"
)

const plugin = "org.example.transcription"

func TestWriteThenReadKeepsTheSegmentsAndTheHash(t *testing.T) {
	root := t.TempDir()
	files, err := textfiles.Stores{}.Open(domain.Vault{ID: "v", Path: root})
	if err != nil {
		t.Fatal(err)
	}
	segments := []domain.TextSegment{{Text: "Good morning.", Selector: `{"t":[0,1.5]}`}}
	hash, err := files.WriteTextFile(t.Context(), plugin, "a.jsonl", segments)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".numen", plugin, "a.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if hash != domain.HashContent(raw) {
		t.Fatalf("hash %s is not the hash of the file", hash)
	}
	got, err := files.ReadTextFile(t.Context(), plugin, "a.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if got.Hash != hash || len(got.Segments) != 1 || got.Segments[0] != segments[0] {
		t.Fatalf("read %+v", got)
	}
	if err := files.DeleteTextFile(t.Context(), plugin, "a.jsonl"); err != nil {
		t.Fatal(err)
	}
	if _, err := files.ReadTextFile(t.Context(), plugin, "a.jsonl"); !errors.Is(err, textport.ErrNoText) {
		t.Fatalf("a deleted file: got %v, want ErrNoText", err)
	}
}

func TestReadTextFileRejectsANameOutsideThePluginFolder(t *testing.T) {
	files, err := textfiles.Stores{}.Open(domain.Vault{ID: "v", Path: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := files.ReadTextFile(t.Context(), plugin, "../core/x.jsonl"); err == nil || errors.Is(err, textport.ErrNoText) {
		t.Fatalf("a name with a slash: got %v, want a refusal", err)
	}
}
