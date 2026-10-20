package file_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/file"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func TestKindsAreAnsweredInTheOrderAskedAndLeaveOutWhatIsNotThere(t *testing.T) {
	v := testsupport.NewVault(t, map[string]string{
		"a.md":   "# a\n",
		"b.mp3":  "sound",
		"c.docx": "left alone",
	})
	index := &typed{types: map[string]domain.NoteType{"a.md": domain.TypeStencil}}

	listing, err := file.NewGetFileKinds(filesystem.VaultReaders{}, index).Execute(
		t.Context(), v, []string{"b.mp3", "missing.md", "../outside.md", "a.md"})
	if err != nil {
		t.Fatal(err)
	}

	if len(listing.Entries) != 2 || listing.Entries[0].Path != "b.mp3" || listing.Entries[1].Path != "a.md" {
		t.Fatalf("kinds are %+v, want b.mp3 then a.md", listing.Entries)
	}
	if listing.Entries[0].Kind != domain.KindRecording || listing.Entries[1].Kind != domain.KindNote {
		t.Errorf("kinds are %v and %v", listing.Entries[0].Kind, listing.Entries[1].Kind)
	}
	if listing.Types["a.md"] != domain.TypeStencil {
		t.Errorf("the note is %v", listing.Types["a.md"])
	}
	if len(index.asked) != 1 || len(index.asked[0]) != 1 {
		t.Errorf("the index was asked %v, want one question about the one note", index.asked)
	}
}
