package source

import (
	"strings"
	"testing"
)

// reader is a Read over a vault whose one document has been read by a model.
func reader(t *testing.T) (Read, *shelf) {
	t.Helper()
	read, _, index, shelved, _ := reading(t, "the words", "outline.pdf")
	if _, err := read.Execute(t.Context(), first, documentPath); err != nil {
		t.Fatal(err)
	}
	return Read{Readers: read.Readers, Sources: index, Derived: shelved}, shelved
}

func TestWhatStandsAfterAChunkIsRead(t *testing.T) {
	u, _ := reader(t)

	whole, err := u.Execute(t.Context(), first, documentPath, 0, MostRead)
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

	whole, err := u.Execute(t.Context(), first, documentPath, 0, MostRead)
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

func TestARunIsCutOnWholeCharacters(t *testing.T) {
	u, shelved := reader(t)
	names := textNames(t, shelved)
	if err := shelved.Write(t.Context(), names.artifact, []byte("\x0c\x0c\nЖивёт слово\n")); err != nil {
		t.Fatal(err)
	}

	// The second byte stands inside Ж, and the run is asked for from there.
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

	if _, err := u.Execute(t.Context(), first, documentPath, 0, MostRead+1); err == nil {
		t.Error("a source was read whole in one question")
	}
	if _, err := u.Execute(t.Context(), first, documentPath, 0, 0); err == nil {
		t.Error("a run of no length was read")
	}
}
