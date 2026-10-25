package source

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/recognition"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// reader is a GetDocumentText over a vault whose one document has been read by a model.
func reader(t *testing.T) (GetDocumentText, *fakeStorage) {
	t.Helper()
	library := newLibrary()
	library.hold(documentPath, domain.KindBook, printPages(outline), 1)
	readers := vaults{first.ID: library}
	index, shelved := newStore(), newShelf()
	read := recognition.NewRecognise(readers, index, fakeStores{shelved}, documents{}, &speaker{says: "the words"})
	read.Batch = 1
	if _, err := read.Execute(t.Context(), first, documentPath); err != nil {
		t.Fatal(err)
	}
	return *NewGetDocumentText(readers, index, fakeStores{shelved}, nil), shelved
}

func TestWhatExistsAfterAChunkIsRead(t *testing.T) {
	u, _ := reader(t)

	whole, err := u.Execute(t.Context(), first, documentPath, 0, MostDocumentText)
	if err != nil {
		t.Fatal(err)
	}
	if whole.Whole == 0 || whole.Text == "" {
		t.Fatalf("the document says nothing: %+v", whole)
	}

	// A chunk that ends part way through, and what stands after it.
	on, err := u.Execute(t.Context(), first, documentPath, 4, whole.Whole)
	if err != nil {
		t.Fatal(err)
	}
	if want := whole.Text[4:]; on.Text != want {
		t.Errorf("read %q from the fifth byte, want %q", on.Text, want)
	}
	if on.Whole != whole.Whole {
		t.Errorf("the source is %d long read from the fifth byte and %d read whole", on.Whole, whole.Whole)
	}
}

func TestARunPastTheEndOfASourceIsNoRun(t *testing.T) {
	u, _ := reader(t)

	whole, err := u.Execute(t.Context(), first, documentPath, 0, MostDocumentText)
	if err != nil {
		t.Fatal(err)
	}
	past, err := u.Execute(t.Context(), first, documentPath, whole.Whole+10, 100)
	if err != nil {
		t.Fatal(err)
	}
	if past.Text != "" || past.Length != 0 {
		t.Errorf("a run past the end read %+v", past)
	}
}

func TestARunIsSplitOnWholeCharacters(t *testing.T) {
	u, shelved := reader(t)
	names := textNames(t, shelved)
	if err := shelved.WriteArtifact(t.Context(), names.artifact, []byte("\x0c\x0c\nЖивёт слово\n")); err != nil {
		t.Fatal(err)
	}

	res, err := u.Execute(t.Context(), first, documentPath, 1, 5)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(res.Text, "Ж") {
		t.Errorf("the run begins %q, want a whole character", res.Text)
	}
	if res.Start != 0 {
		t.Errorf("the run begins at %d, want the character's own byte", res.Start)
	}
}

func TestAskingForMoreThanOneQuestionCarriesIsRefused(t *testing.T) {
	u, _ := reader(t)

	if _, err := u.Execute(t.Context(), first, documentPath, 0, MostDocumentText+1); err == nil {
		t.Error("a source was read whole in one question")
	}
	if _, err := u.Execute(t.Context(), first, documentPath, 0, 0); err == nil {
		t.Error("a run of no length was read")
	}
}

// A run's bounds stand on whole characters wherever it begins and however
// long it is asked for, including where it begins before the text, past it,
// or asks for nothing at all.
func TestGetRuneBounds(t *testing.T) {
	// é is two bytes, so a run touching it from either side relies on it
	// whole or not at all.
	const prose = "éllo"

	cases := []struct {
		name                  string
		start, length         int
		wantStart, wantLength int
	}{
		{"a run past the end is no run", 10, 3, len(prose), 0},
		{"a run is cut on whole characters", 1, 3, 0, 4},
		{"a negative start begins at the start of the text", -3, 2, 0, 2},
		{"a negative start still begins on a whole character", -5, 1, 0, 2},
		{"a run of zero length is empty", 2, 0, 2, 0},
		{"a run of negative length is empty", 2, -4, 2, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			start, length := getRuneBounds(prose, c.start, c.length)
			if start != c.wantStart || length != c.wantLength {
				t.Errorf("getRuneBounds(%q, %d, %d) = (%d, %d), want (%d, %d)",
					prose, c.start, c.length, start, length, c.wantStart, c.wantLength)
			}
		})
	}
}
