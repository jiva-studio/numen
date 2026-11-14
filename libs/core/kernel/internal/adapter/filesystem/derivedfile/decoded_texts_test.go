package derivedfile_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
)

func readText(t *testing.T, stores derivedfile.TextStores, v domain.Vault, name string) text.File {
	t.Helper()
	files, err := stores.Open(v)
	if err != nil {
		t.Fatal(err)
	}
	file, err := files.ReadTextFile(t.Context(), transcriber, name)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

// A file read through the decoded texts is the file read without them, on the first read and on the next.
func TestDecodedTextsReturnTheContentOfTheFile(t *testing.T) {
	v := domain.Vault{ID: "v", Path: t.TempDir()}
	plain := derivedfile.TextStores{}
	kept := derivedfile.TextStores{Decoded: derivedfile.NewDecodedTexts(derivedfile.DefaultDecodedBudget)}
	files, err := plain.Open(v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := files.WriteTextFile(t.Context(), transcriber, "lecture.jsonl", []domain.TextSegment{
		{Text: "The river runs to the sea.\n", Heading: "Rivers", Selector: `{"t":[0,2.5]}`},
		{Text: "The wind blows over it.", Selector: `{"t":[2.5,4]}`},
	}); err != nil {
		t.Fatal(err)
	}

	want := readText(t, plain, v, "lecture.jsonl")
	for _, read := range []string{"first", "second"} {
		got := readText(t, kept, v, "lecture.jsonl")
		if got.TextHash != want.TextHash || !slices.Equal(got.Segments, want.Segments) {
			t.Fatalf("the %s read is %+v, want %+v", read, got, want)
		}
	}
}

// A file whose content changed on disk is decoded again, with the hash of its new content.
func TestTheDecodedTextsReadAChangedFileAgain(t *testing.T) {
	root := t.TempDir()
	v := domain.Vault{ID: "v", Path: root}
	kept := derivedfile.TextStores{Decoded: derivedfile.NewDecodedTexts(derivedfile.DefaultDecodedBudget)}
	files, err := kept.Open(v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := files.WriteTextFile(t.Context(), transcriber, "lecture.jsonl", []domain.TextSegment{{Text: "Good morning."}}); err != nil {
		t.Fatal(err)
	}
	before := readText(t, kept, v, "lecture.jsonl")

	changed := []byte(`{"text":"Good evening."}` + "\n")
	if err := os.WriteFile(filepath.Join(root, ".numen", transcriber, "lecture.jsonl"), changed, 0o600); err != nil {
		t.Fatal(err)
	}
	after := readText(t, kept, v, "lecture.jsonl")

	if after.TextHash != domain.HashContent(changed) || after.TextHash == before.TextHash {
		t.Fatalf("the hash after the change is %s, want %s", after.TextHash, domain.HashContent(changed))
	}
	if len(after.Segments) != 1 || after.Segments[0].Text != "Good evening." {
		t.Fatalf("the segments after the change are %+v", after.Segments)
	}
}
