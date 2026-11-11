package file_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestGetFileKinds_PreservesRequestOrderAndOmitsMissing(t *testing.T) {
	v := vaultfixture.NewVault(t, map[string]string{
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

func TestGetFileKinds_CarriesTheFrontmatterOfEachNote(t *testing.T) {
	v := vaultfixture.NewVault(t, map[string]string{
		"links/tide.md": "---\ntype: link\nurl: https://tides.example/mill\nmedia: plugin://org.example.links/tide.mp4\n---\n",
		"plain.md":      "# plain\n",
		"b.mp3":         "sound",
	})
	index := &typed{types: map[string]domain.NoteType{"links/tide.md": "link", "plain.md": domain.TypeNote}}

	listing, err := file.NewGetFileKinds(vaultreader.Readers{}, index).Execute(
		t.Context(), v, []string{"links/tide.md", "plain.md", "b.mp3"})
	if err != nil {
		t.Fatal(err)
	}

	tide := listing.Properties["links/tide.md"]
	if tide["url"] != "https://tides.example/mill" || tide["media"] != "plugin://org.example.links/tide.mp4" {
		t.Errorf("the link note carries %v", tide)
	}
	if _, isListed := listing.Properties["b.mp3"]; isListed {
		t.Error("a recording carries properties")
	}
}

func TestGetFileKinds_UnindexedNoteUsesFileNameType(t *testing.T) {
	v := vaultfixture.NewVault(t, map[string]string{
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
