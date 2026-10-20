package source

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func TestDocumentsAreListedWithWhetherAModelReadThem(t *testing.T) {
	index := newStore()
	index.sources[first.ID] = map[string]domain.Source{
		"library/scan.pdf":  {Fingerprint: domain.Fingerprint{Path: "library/scan.pdf", Kind: domain.KindBook}, Producer: "ocr"},
		"library/typed.pdf": {Fingerprint: domain.Fingerprint{Path: "library/typed.pdf", Kind: domain.KindBook}},
		"note.md":           {Fingerprint: domain.Fingerprint{Path: "note.md", Kind: domain.KindNote}},
	}

	got, err := NewGetDocumentList(index).Execute(t.Context(), first)
	if err != nil {
		t.Fatal(err)
	}
	want := []Document{{Path: "library/scan.pdf", IsRead: true}, {Path: "library/typed.pdf"}}
	if !slices.Equal(got, want) {
		t.Errorf("listed %+v, want %+v", got, want)
	}
}

func TestAVaultWithNoDocumentsListsNone(t *testing.T) {
	got, err := NewGetDocumentList(newStore()).Execute(t.Context(), second)
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("listed %+v, %v", got, err)
	}
}
