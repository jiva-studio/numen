package service_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
)

// The invented shelves: "Bücher" with its ü as one character (NFC) as the vault stores it, and as u
// followed by a combining diaeresis (NFD) as some keyboards and file systems write it.
const (
	shelfNFC = "Bücher"
	shelfNFD = "Bücher"
)

// shelfPassages are the passages of the invented shelves: library has one, library2 has one whose
// path starts with the letters of library, and the shelf of books has one.
func shelfPassages() []host.Passage {
	return []host.Passage{
		{Path: "library/Tea.pdf", Text: "green tea leaves", Start: 0, Length: 16},
		{Path: "library2/Tea Atlas.pdf", Text: "tea gardens of the hills", Start: 0, Length: 24},
		{Path: shelfNFC + "/Teebuch.pdf", Text: "black tea from the hills", Start: 0, Length: 24},
	}
}

func TestLibrarySearchSendsEveryFolderPrefixFormWithASlash(t *testing.T) {
	calls, registry := newRecordedSearch(t, hosttest.New(testVault), shelfPassages())

	for _, prefix := range []string{"./library", "/library/", "library", "library/"} {
		var out librarySearchAnswer
		if failure := callFailure(t, registry, testVault, "library_search", `{"query":"tea","pathPrefix":"`+prefix+`"}`, &out); failure != "" {
			t.Fatalf("library_search with the prefix %q rejected: %s", prefix, failure)
		}
		queries := calls.getQueries()
		if got := queries[len(queries)-1].PathPrefix; got != "library/" {
			t.Errorf("the prefix %q was sent to the host as %q, want library/", prefix, got)
		}
		if got := getMatchPaths(out); !slices.Equal(got, []string{"library/Tea.pdf"}) {
			t.Errorf("the prefix %q found %v, want library/Tea.pdf alone, not library2", prefix, got)
		}
	}
}

func TestLibrarySearchSendsADecomposedPrefixComposed(t *testing.T) {
	calls, registry := newRecordedSearch(t, hosttest.New(testVault), shelfPassages())

	var out librarySearchAnswer
	if failure := callFailure(t, registry, testVault, "library_search", `{"query":"tea","pathPrefix":"`+shelfNFD+`"}`, &out); failure != "" {
		t.Fatal(failure)
	}
	queries := calls.getQueries()
	if len(queries) != 1 || queries[0].PathPrefix != shelfNFC+"/" {
		t.Errorf("the decomposed prefix was sent to the host as %+v, want %q", queries, shelfNFC+"/")
	}
	if got := getMatchPaths(out); !slices.Equal(got, []string{shelfNFC + "/Teebuch.pdf"}) {
		t.Errorf("the decomposed prefix found %v, want the book of the shelf", got)
	}
}

func TestLibrarySearchWithoutAPrefixSearchesTheWholeVault(t *testing.T) {
	calls, registry := newRecordedSearch(t, hosttest.New(testVault), shelfPassages())

	for _, prefix := range []string{"", "./", "/"} {
		var out librarySearchAnswer
		if failure := callFailure(t, registry, testVault, "library_search", `{"query":"tea","pathPrefix":"`+prefix+`"}`, &out); failure != "" {
			t.Fatal(failure)
		}
		queries := calls.getQueries()
		if got := queries[len(queries)-1].PathPrefix; got != "" {
			t.Errorf("the prefix %q was sent to the host as %q, want none", prefix, got)
		}
		if len(out.Matches) != 3 {
			t.Errorf("the prefix %q found %v, want every passage", prefix, getMatchPaths(out))
		}
	}
}

// newShelfNotes is a vault of invented notes in the folders notes, notes2 and the shelf of books.
func newShelfNotes(t *testing.T) *hosttest.Vault {
	t.Helper()
	v := hosttest.New(testVault)
	v.File("notes/Tea.md", "---\ntitle: Tea\n---\nGreen tea.")
	v.File("notes2/Coffee.md", "---\ntitle: Coffee\n---\nDark roast.")
	v.File(shelfNFC+"/Lied.md", "---\ntitle: Lied\n---\nA song.")
	v.File("Index.md", "---\ntitle: Index\n---\nEvery shelf.")
	return v
}

func TestNoteListAcceptsAFolderWithADotSlashOrSlashes(t *testing.T) {
	v := newShelfNotes(t)
	_, registry := newRecordedSearch(t, v, nil)

	for _, folder := range []string{"./notes", "/notes/", "notes", "notes/"} {
		var out noteListOutput
		if failure := callFailure(t, registry, testVault, "note_list", `{"folder":"`+folder+`"}`, &out); failure != "" {
			t.Fatalf("note_list of the folder %q rejected: %s", folder, failure)
		}
		if got := getPaths(out.Notes); !slices.Equal(got, []string{"notes/Tea.md"}) {
			t.Errorf("note_list of the folder %q listed %v, want notes/Tea.md alone, not notes2", folder, got)
		}
	}
}

func TestNoteListFindsAComposedFolderGivenDecomposed(t *testing.T) {
	v := newShelfNotes(t)
	_, registry := newRecordedSearch(t, v, nil)

	var out noteListOutput
	if failure := callFailure(t, registry, testVault, "note_list", `{"folder":"`+shelfNFD+`"}`, &out); failure != "" {
		t.Fatal(failure)
	}
	if got := getPaths(out.Notes); !slices.Equal(got, []string{shelfNFC + "/Lied.md"}) {
		t.Errorf("note_list of the decomposed folder listed %v, want the note of the shelf", got)
	}
}
