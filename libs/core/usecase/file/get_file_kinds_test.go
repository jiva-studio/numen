package file_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestGetFileKinds_PreservesRequestOrderAndOmitsMissing(t *testing.T) {
	v := testsupport.NewVault(t, map[string]string{
		"a.md":   "# a\n",
		"b.mp3":  "sound",
		"c.docx": "left alone",
	})
	index := &typed{types: map[string]domain.NoteType{"a.md": "form"}}

	listing, err := file.NewGetFileKinds(vaultreader.Readers{}, index).Execute(
		t.Context(), v, []string{"b.mp3", "missing.md", "../outside.md", "a.md"})
	if err != nil {
		t.Fatal(err)
	}

	if len(listing.Entries) != 2 || listing.Entries[0].Path != "b.mp3" || listing.Entries[1].Path != "a.md" {
		t.Fatalf("kinds are %+v, want b.mp3 then a.md", listing.Entries)
	}
	if listing.Entries[0].Kind != domain.KindFile || listing.Entries[1].Kind != domain.KindNote {
		t.Errorf("kinds are %v and %v", listing.Entries[0].Kind, listing.Entries[1].Kind)
	}
	if listing.Types["a.md"] != "form" {
		t.Errorf("the note is %v", listing.Types["a.md"])
	}
	if len(index.requested) != 1 || len(index.requested[0]) != 1 {
		t.Errorf("the index was queried %v, want one question about the one note", index.requested)
	}
}

func TestGetFileKinds_UnindexedNoteUsesFileNameType(t *testing.T) {
	v := testsupport.NewVault(t, map[string]string{
		"presets/gentle.md": "---\ntype: preset\n---\n# Gentle\n",
		"plain.md":          "# plain\n",
	})
	index := &typed{types: map[string]domain.NoteType{}}

	listing, err := file.NewGetFileKinds(vaultreader.Readers{}, index).Execute(
		t.Context(), v, []string{"presets/gentle.md", "plain.md"})
	if err != nil {
		t.Fatal(err)
	}

	if listing.Types["presets/gentle.md"] != "preset" {
		t.Errorf("the new preset is %q, want preset", listing.Types["presets/gentle.md"])
	}
	if listing.Types["plain.md"] != domain.TypeNote {
		t.Errorf("the plain note is %q, want %q", listing.Types["plain.md"], domain.TypeNote)
	}
}
