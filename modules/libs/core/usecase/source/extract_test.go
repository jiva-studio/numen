package source

import (
	"errors"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/chunking"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/epub"
	"github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

const bookPath = "library/book.epub"

func TestABookIsCutIntoChunksRecordedWithTheRecipeThatCutThem(t *testing.T) {
	cases := []struct {
		name  string
		sizes chunking.Sizes
	}{
		{"at the sizes configuration names none for", chunking.Sizes{}},
		{"at sizes of its own", chunking.Sizes{Large: 60, LargeOverlap: 10, Small: 20, SmallOverlap: 5}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := t.Context()
			index, shelf := newStore(), newLibrary()
			raw := bookOf(t, "A Book", words(sanskrit, 400), words(sanskrit, 400))
			shelf.hold(bookPath, domain.KindBook, raw, 1)

			extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Known: index, Sizes: c.sizes}
			res, err := extract.Execute(ctx, first)
			if err != nil {
				t.Fatal(err)
			}
			if res.Seen != 1 || res.Recorded != 1 || res.Extracted != 1 || res.Unreadable != 0 {
				t.Errorf("the run reports %+v, want one book found, recorded and extracted", res)
			}
			if res.Chunks != len(index.chunks) {
				t.Errorf("the run counted %d chunks and wrote %d", res.Chunks, len(index.chunks))
			}
			if index.written[bookPath] != 1 {
				t.Errorf("the source was written %d times, want one write", index.written[bookPath])
			}

			source := index.sources[first.ID][bookPath]
			if want := recipe(text.ReaderEPUB, extract.sizes()); source.Recipe != want {
				t.Errorf("recipe = %q, want %q", source.Recipe, want)
			}
			if source.Hash == "" {
				t.Error("nothing addresses the content the chunks were cut from")
			}
			if source.Fingerprint.Size != int64(len(raw)) ||
				!source.Fingerprint.ModTime.Equal(time.Unix(0, 1)) {
				t.Errorf("the source was recorded as %+v, want the file as it is", source.Fingerprint)
			}

			// A chunk keeps a place in the extracted text and not the text
			// itself, so what it says has to be what stands at that place.
			book, err := epub.Read(raw)
			if err != nil {
				t.Fatal(err)
			}
			large, small, located := 0, 0, 0
			for _, c := range index.chunks {
				if c.start < 0 || c.start+c.length > len(book.Text) {
					t.Fatalf("chunk %d runs outside the book's text", c.id)
				}
				if got := book.Text[c.start : c.start+c.length]; got != c.text {
					t.Errorf("chunk %d says %q and its place holds %q", c.id, c.text, got)
				}
				if c.parent == 0 {
					large++
				} else {
					small++
				}
				if strings.Contains(c.location, "Part 2") {
					located++
				}
			}
			if large == 0 || small == 0 {
				t.Errorf("%d large chunks and %d small ones, want both sizes", large, small)
			}
			if located == 0 {
				t.Error("no chunk is where the book says it is: nothing carries the name of a part")
			}
		})
	}
}

func TestABookThatWillNotParseDoesNotStopTheOthers(t *testing.T) {
	ctx := t.Context()
	index, shelf := newStore(), newLibrary()

	// The unreadable one sorts between the two books, so a run that stops at it
	// leaves the second uncut.
	shelf.hold("library/a.epub", domain.KindBook, bookOf(t, "A", words(sanskrit, 200)), 1)
	shelf.hold("library/broken.epub", domain.KindBook, []byte("this is not an archive at all"), 1)
	shelf.hold("library/z.epub", domain.KindBook, bookOf(t, "Z", words(sanskrit, 200)), 1)
	shelf.hold("notes/one.md", domain.KindNote, []byte("# One\n\nA note.\n"), 1)

	res, err := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Known: index}.Execute(ctx, first)
	if err != nil {
		t.Fatalf("one book that will not parse ended the run: %v", err)
	}
	if res.Seen != 3 {
		t.Errorf("seen = %d, want the three books and not the note", res.Seen)
	}
	if res.Extracted != 2 || res.Unreadable != 1 {
		t.Errorf("the run reports %+v, want two books extracted and one unreadable", res)
	}
	for _, path := range []string{"library/a.epub", "library/z.epub"} {
		if !index.cut(first.ID, path) {
			t.Errorf("%s was not cut", path)
		}
	}
	if index.cut(first.ID, "library/broken.epub") {
		t.Error("something that is not a book was cut into chunks")
	}
	if got := shelf.reads["library/broken.epub"]; got != 1 {
		t.Errorf("the unreadable book was read %d times, want one attempt in a run", got)
	}
	broken, known := index.sources[first.ID]["library/broken.epub"]
	if !known {
		t.Error("the file the vault holds is not a source the index knows about")
	}
	if broken.Recipe != "" {
		t.Errorf("a file nothing could read claims to have been cut by %q", broken.Recipe)
	}
	if index.cut(first.ID, "notes/one.md") {
		t.Error("a note was cut by the pass over books")
	}
}

func TestWhatHasNotChangedIsNotOpenedAgain(t *testing.T) {
	ctx := t.Context()
	index, shelf := newStore(), newLibrary()
	shelf.hold(bookPath, domain.KindBook, bookOf(t, "A Book", words(sanskrit, 300)), 1)

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Known: index}
	if _, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}
	reads := shelf.reads[bookPath]

	res, err := extract.Execute(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if res.Unchanged != 1 || res.Recorded != 0 || res.Extracted != 0 {
		t.Errorf("the second run reports %+v, want the book skipped on its fingerprint", res)
	}
	if shelf.reads[bookPath] != reads {
		t.Errorf("the file was read %d times over two runs, want %d", shelf.reads[bookPath], reads)
	}
	if index.written[bookPath] != 1 {
		t.Errorf("the source was written %d times, want the one write", index.written[bookPath])
	}

	shelf.hold(bookPath, domain.KindBook, bookOf(t, "A Book", words(sanskrit, 400)), 2)
	res, err = extract.Execute(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if res.Recorded != 1 || res.Extracted != 1 {
		t.Errorf("the run after the file changed reports %+v, want it read again", res)
	}
	if index.written[bookPath] != 2 {
		t.Errorf("the source was written %d times, want a second write", index.written[bookPath])
	}
}

// An overlap as wide as the size it belongs to is narrowed before anything is
// cut, so that tiling advances. A recipe naming the number configuration asked
// for would name bytes nobody wrote, and the source would be read again on
// every run.
func TestTheRecipeNamesTheSizesTheCutKeptTo(t *testing.T) {
	asked := chunking.Sizes{Large: 100, LargeOverlap: 100, Small: 20, SmallOverlap: 20}
	kept := asked.Resolved()

	if kept.LargeOverlap != 99 || kept.SmallOverlap != 19 {
		t.Fatalf("the cut keeps to %+v", kept)
	}
	if got := (Extract{Sizes: asked}).sizes(); got != kept {
		t.Errorf("the recipe names %+v, want %+v", got, kept)
	}
}

func TestTheRecipeFollowsTheCutSizesAndStalenessFollowsTheRecipe(t *testing.T) {
	ctx := t.Context()
	one := chunking.Sizes{Large: 100, LargeOverlap: 20, Small: 20, SmallOverlap: 5}
	other := one
	other.Small = 21

	if recipe(text.ReaderEPUB, one) == recipe(text.ReaderEPUB, other) {
		t.Fatal("one word of difference in a chunk size is not in the recipe")
	}

	index, shelf := newStore(), newLibrary()
	shelf.hold(bookPath, domain.KindBook, bookOf(t, "A Book", words(sanskrit, 300)), 1)

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Known: index, Sizes: one}
	if _, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}

	stale, err := index.ByOtherRecipe(ctx, first.ID, domain.KindBook, recipes(Extract{Sizes: other}.sizes()), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 1 || stale[0] != bookPath {
		t.Errorf("%v were cut by another recipe, want the one book", stale)
	}

	extract.Sizes = other
	res, err := extract.Execute(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if res.Extracted != 1 || res.Recorded != 0 {
		t.Errorf("the run at other sizes reports %+v, want the book cut again and nothing recorded", res)
	}
	if index.written[bookPath] != 2 {
		t.Errorf("the source was written %d times, want the cut at each of the two sizes", index.written[bookPath])
	}
	if want := recipe(text.ReaderEPUB, extract.sizes()); index.sources[first.ID][bookPath].Recipe != want {
		t.Errorf("recipe = %q, want %q", index.sources[first.ID][bookPath].Recipe, want)
	}

	if _, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}
	if index.written[bookPath] != 2 {
		t.Errorf("a run at the sizes already used wrote the source again, %d writes in all", index.written[bookPath])
	}
}

func TestExtractionStaysInsideItsVault(t *testing.T) {
	ctx := t.Context()
	index := newStore()

	shelves := vaults{first.ID: newLibrary(), second.ID: newLibrary()}
	shelves[first.ID].hold("library/sanskrit.epub", domain.KindBook,
		bookOf(t, "Sanskrit", words(sanskrit, 300), words(sanskrit, 300)), 1)
	shelves[second.ID].hold("library/latin.epub", domain.KindBook,
		bookOf(t, "Latin", words(latin, 300), words(latin, 300)), 1)

	extract := Extract{Readers: shelves, Sources: index, Known: index}
	for _, v := range []domain.Vault{first, second} {
		if _, err := extract.Execute(ctx, v); err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		vault  domain.Vault
		path   string
		holds  string
		absent string
	}{
		{first, "library/sanskrit.epub", sanskrit[0], latin[0]},
		{second, "library/latin.epub", latin[0], sanskrit[0]},
	}
	for _, c := range cases {
		t.Run(c.vault.Name, func(t *testing.T) {
			chunks := index.small(c.vault.ID)
			if len(chunks) == 0 {
				t.Fatalf("%s holds no chunk that carries a vector", c.vault.Name)
			}
			for _, chunk := range chunks {
				if chunk.path != c.path {
					t.Fatalf("%s holds a chunk of %s", c.vault.Name, chunk.path)
				}
				if !strings.Contains(chunk.text, c.holds) {
					t.Fatalf("a chunk of %s is not written in its words: %q", c.vault.Name, chunk.text)
				}
				if strings.Contains(chunk.text, c.absent) {
					t.Fatalf("a chunk of %s holds the other vault's words: %q", c.vault.Name, chunk.text)
				}
			}
			known, err := index.Fingerprints(ctx, c.vault.ID, domain.KindBook)
			if err != nil {
				t.Fatal(err)
			}
			if len(known) != 1 || known[c.path].Path != c.path {
				t.Errorf("%s knows about %v, want its own book alone", c.vault.Name, known)
			}
		})
	}
}

// Removal takes a book's source row and every chunk cut from it out of the
// index. The text of a passage is read from its file, so a passage naming a file
// that is gone can answer nothing.
func TestABookTheVaultNoLongerHoldsIsTakenOut(t *testing.T) {
	ctx := t.Context()
	index, shelf := newStore(), newLibrary()

	shelf.hold("library/kept.epub", domain.KindBook, bookOf(t, "Kept", words(sanskrit, 200)), 1)
	shelf.hold("library/gone.epub", domain.KindBook, bookOf(t, "Gone", words(latin, 200)), 1)

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Known: index}
	if _, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}
	if !index.cut(first.ID, "library/gone.epub") {
		t.Fatal("the book was not cut, so its removal proves nothing")
	}

	delete(shelf.files, "library/gone.epub")

	res, err := extract.Execute(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if res.Removed != 1 {
		t.Errorf("removed = %d, want the one book the vault no longer holds", res.Removed)
	}
	if _, known := index.sources[first.ID]["library/gone.epub"]; known {
		t.Error("a book the vault does not hold is still a source the index knows about")
	}
	if index.cut(first.ID, "library/gone.epub") {
		t.Error("the chunks of a book that is gone are still in the index")
	}

	// The one that stayed is untouched: removal names paths.
	if _, known := index.sources[first.ID]["library/kept.epub"]; !known {
		t.Error("the book that is still there was taken out as well")
	}
	if !index.cut(first.ID, "library/kept.epub") {
		t.Error("the chunks of the book that is still there were taken out as well")
	}
}

// A file whose content changed while its size and modification time did not is
// skipped: the index believes those three and nothing else. An archive restored
// by `unzip` and a tree brought over by `rsync -tc` both do that, and the chunks
// then describe text the file no longer has. Rebuilding is the way out.
func TestRebuildingTheIndexReadsEveryFile(t *testing.T) {
	ctx := t.Context()
	index, shelf := newStore(), newLibrary()

	was := words(latin, 200)
	// Other words, letter for letter, so the size is the same one.
	now := strings.Map(func(r rune) rune {
		if r == 'a' {
			return 'e'
		}
		return r
	}, was)
	before, after := bookOf(t, "A", was), bookOf(t, "A", now)
	if len(before) != len(after) || string(before) == string(after) {
		t.Fatalf("the two books are %d and %d bytes, and this asks about equal ones",
			len(before), len(after))
	}

	shelf.hold("library/A.epub", domain.KindBook, before, 1)

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Known: index}
	if res, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	} else if res.Extracted != 1 {
		t.Fatalf("read %d books", res.Extracted)
	}

	shelf.hold("library/A.epub", domain.KindBook, after, 1)

	if res, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	} else if res.Unchanged != 1 || res.Extracted != 0 {
		t.Errorf("the fingerprint is the path, the size and the time: %+v", res)
	}

	extract.RebuildIndex = true
	if res, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	} else if res.Extracted != 1 {
		t.Errorf("rebuilding read %d books: %+v", res.Extracted, res)
	}
}

// A recognition still running is the source's text while it runs. The pages it
// has read are what a passage is a place in, and the artifact answers under the
// same producer name once the run finishes.
func TestASourceCutFromAPartialReadsBackFromThePartial(t *testing.T) {
	ctx := t.Context()
	index, shelf, made := newStore(), newLibrary(), newShelf()

	raw := bookOf(t, "A Book", words(sanskrit, 200))
	shelf.hold(bookPath, domain.KindBook, raw, 1)

	const (
		read  = "What the pages read so far say."
		whole = "What every page of the document says."
	)
	hash := text.Fingerprint(raw)
	if err := made.Write(ctx, text.Partial("ocr", hash), []byte(read)); err != nil {
		t.Fatal(err)
	}

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Known: index, Derived: made}
	if _, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}
	src := index.sources[first.ID][bookPath]
	if src.Producer != "ocr" {
		t.Fatalf("the source names %q as the producer of its text, want ocr", src.Producer)
	}

	of := text.Reader{Vault: shelf, Derived: made}
	doc, err := of.Of(ctx, bookPath, src.Producer, src.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Text != read {
		t.Errorf("the source reads back as %q, want the pages the run has read", doc.Text)
	}

	if err := made.Write(ctx, text.Artifact("ocr", hash), []byte(whole)); err != nil {
		t.Fatal(err)
	}
	doc, err = of.Of(ctx, bookPath, src.Producer, src.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Text != whole {
		t.Errorf("the source reads back as %q, want the finished artifact", doc.Text)
	}
}

// A source the vault no longer holds takes the files of its reading with it.
// One run made them and none of them means anything without the others.
func TestABookTakenOutTakesTheFilesOfItsReading(t *testing.T) {
	ctx := t.Context()
	index, shelf, made := newStore(), newLibrary(), newShelf()

	gone := bookOf(t, "Gone", words(latin, 200))
	kept := bookOf(t, "Kept", words(sanskrit, 200))
	shelf.hold("library/gone.epub", domain.KindBook, gone, 1)
	shelf.hold("library/kept.epub", domain.KindBook, kept, 1)

	for _, raw := range [][]byte{gone, kept} {
		for _, name := range text.Names("ocr", text.Fingerprint(raw)) {
			if err := made.Write(ctx, name, []byte("What the pages say.")); err != nil {
				t.Fatal(err)
			}
		}
	}

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Known: index, Derived: made}
	if _, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}
	if index.sources[first.ID]["library/gone.epub"].Producer == "" {
		t.Fatal("the book was not cut from its reading, so its removal proves nothing")
	}

	delete(shelf.files, "library/gone.epub")
	res, err := extract.Execute(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if res.Removed != 1 {
		t.Fatalf("removed = %d, want the one book the vault no longer holds", res.Removed)
	}

	for _, name := range text.Names("ocr", text.Fingerprint(gone)) {
		if _, err := made.Read(ctx, name); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s is still in the store", name)
		}
	}
	for _, name := range text.Names("ocr", text.Fingerprint(kept)) {
		if _, err := made.Read(ctx, name); err != nil {
			t.Errorf("%s went with the other book: %v", name, err)
		}
	}
}

func TestARenamedBookKeepsItsReading(t *testing.T) {
	// A reading is named by the hash of what was read, so a document renamed is
	// the same reading at another path. Renaming is a path gone and a path
	// found, and an hour of a model's work stands on the one that went.
	ctx := t.Context()
	index, shelf, made := newStore(), newLibrary(), newShelf()

	raw := bookOf(t, "Read", words(sanskrit, 200))
	shelf.hold("library/before.epub", domain.KindBook, raw, 1)
	for _, name := range text.Names("ocr", text.Fingerprint(raw)) {
		if err := made.Write(ctx, name, []byte("What the pages say.")); err != nil {
			t.Fatal(err)
		}
	}

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Known: index, Derived: made}
	if _, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}
	if index.sources[first.ID]["library/before.epub"].Producer == "" {
		t.Fatal("the book was not cut from its reading, so renaming it proves nothing")
	}

	delete(shelf.files, "library/before.epub")
	shelf.hold("library/after.epub", domain.KindBook, raw, 1)
	if _, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}

	for _, name := range text.Names("ocr", text.Fingerprint(raw)) {
		if _, err := made.Read(ctx, name); err != nil {
			t.Errorf("renaming the book threw away %s: %v", name, err)
		}
	}
	if from := index.sources[first.ID]["library/after.epub"].Producer; from == "" {
		t.Error("the renamed book does not stand on its reading")
	}
}

func TestOneOfTwoCopiesTakenOutLeavesTheOtherReading(t *testing.T) {
	// Two copies of one document are one reading, because the name is the hash
	// of the bytes. Taking one copy out leaves the other standing on it.
	ctx := t.Context()
	index, shelf, made := newStore(), newLibrary(), newShelf()

	raw := bookOf(t, "Twice", words(sanskrit, 200))
	shelf.hold("library/one.epub", domain.KindBook, raw, 1)
	shelf.hold("shelf/two.epub", domain.KindBook, raw, 1)
	for _, name := range text.Names("ocr", text.Fingerprint(raw)) {
		if err := made.Write(ctx, name, []byte("What the pages say.")); err != nil {
			t.Fatal(err)
		}
	}

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Known: index, Derived: made}
	if _, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}

	delete(shelf.files, "library/one.epub")
	if _, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}

	for _, name := range text.Names("ocr", text.Fingerprint(raw)) {
		if _, err := made.Read(ctx, name); err != nil {
			t.Errorf("one copy going took %s with it: %v", name, err)
		}
	}
	if from := index.sources[first.ID]["shelf/two.epub"].Producer; from == "" {
		t.Error("the copy that stayed lost its reading")
	}
}

const talkPath = "talks/a lecture.mp3"

// transcribed is the transcript a model leaves of one talk.
func transcribed() []byte {
	return transcript.Marshal([]transcript.Cue{
		{Text: words(sanskrit, 200), From: 1500, To: 5025000},
		{Text: words(latin, 200), From: 5025000, To: 5400000},
	})
}

// A recording nobody has listened to is a source found by its name, with
// nothing to cut into chunks.
func TestARecordingNobodyHasHeardIsASourceWithNoChunks(t *testing.T) {
	ctx := t.Context()
	index, shelf := newStore(), newLibrary()
	shelf.hold(talkPath, domain.KindRecording, []byte("ID3 and then the samples"), 1)

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Known: index}
	res, err := extract.Execute(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if res.Seen != 1 || res.Recorded != 1 {
		t.Errorf("the run reports %+v, want the recording found and recorded", res)
	}
	if res.Chunks != 0 || res.Extracted != 0 {
		t.Errorf("the run cut %d chunks out of a recording nobody has heard", res.Chunks)
	}
	if _, held := index.sources[first.ID][talkPath]; !held {
		t.Error("the vault holds a recording the index does not")
	}
}

// What a model heard is the text of the recording, and a chunk of it is located
// by when it was said.
func TestARecordingIsCutFromWhatWasHeardInIt(t *testing.T) {
	ctx := t.Context()
	index, shelf, made := newStore(), newLibrary(), newShelf()

	raw := []byte("ID3 and then the samples")
	shelf.hold(talkPath, domain.KindRecording, raw, 1)
	if err := made.Write(ctx, text.Artifact(text.ASR, text.Fingerprint(raw)), transcribed()); err != nil {
		t.Fatal(err)
	}

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Known: index, Derived: made}
	res, err := extract.Execute(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if res.Extracted != 1 || res.Chunks == 0 {
		t.Fatalf("the run reports %+v, want the recording cut into chunks", res)
	}

	src := index.sources[first.ID][talkPath]
	if src.Producer != text.ASR {
		t.Errorf("the source names %q as the producer of its text, want %q", src.Producer, text.ASR)
	}
	if want := recipe(text.ReaderRecording, extract.sizes()); src.Recipe != want {
		t.Errorf("recipe = %q, want %q", src.Recipe, want)
	}
	if !slices.Contains(recipes(extract.sizes()), src.Recipe) {
		t.Error("the recipe a recording is cut by is not one the next run knows")
	}

	for _, c := range index.chunks {
		if c.location == "" {
			t.Fatal("a chunk of a recording says nothing about where it is")
		}
	}
}

// A transcription writes a transcript and the note of what heard it, and a
// recording the vault no longer holds takes both with it.
func TestARecordingTakenOutTakesTheFilesOfItsTranscription(t *testing.T) {
	ctx := t.Context()
	index, shelf, made := newStore(), newLibrary(), newShelf()

	raw := []byte("ID3 and then the samples")
	shelf.hold(talkPath, domain.KindRecording, raw, 1)
	hash := text.Fingerprint(raw)
	if err := made.Write(ctx, text.Artifact(text.ASR, hash), transcribed()); err != nil {
		t.Fatal(err)
	}
	if err := made.Write(ctx, text.Beside(text.ASR, hash), []byte(`{"model":"parakeet"}`)); err != nil {
		t.Fatal(err)
	}

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Known: index, Derived: made}
	if _, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}

	delete(shelf.files, talkPath)
	res, err := extract.Execute(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if res.Removed != 1 {
		t.Fatalf("removed = %d, want the one recording the vault no longer holds", res.Removed)
	}
	if left := made.names(); len(left) != 0 {
		t.Errorf("the store still holds %v", left)
	}
}

// A vault already read is not read again: what a book is cut by, and where its
// reading is kept, are what they were before recordings were a kind of their
// own.
func TestABookIsCutTheWayItAlwaysWas(t *testing.T) {
	ctx := t.Context()
	index, shelf := newStore(), newLibrary()
	raw := bookOf(t, "A Book", words(sanskrit, 400))
	shelf.hold(bookPath, domain.KindBook, raw, 1)
	shelf.hold(talkPath, domain.KindRecording, []byte("ID3 and then the samples"), 1)

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Known: index}
	if _, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}

	sizes := extract.sizes()
	if want := "epub-1/large=200+40/small=50+10/limit=1000"; recipe(text.ReaderEPUB, sizes) != want {
		t.Errorf("a book is cut by %q, want %q", recipe(text.ReaderEPUB, sizes), want)
	}
	if got := index.sources[first.ID][bookPath].Recipe; got != recipe(text.ReaderEPUB, sizes) {
		t.Errorf("the book was cut by %q", got)
	}
	if got := extract.producer(domain.Fingerprint{Kind: domain.KindBook}); got != "ocr" {
		t.Errorf("a book's reading is kept under %q, want ocr", got)
	}
	if got := text.Artifact("ocr", "abc123"); got != "ocr/abc123.txt" {
		t.Errorf("a reading is kept under %q", got)
	}
}
