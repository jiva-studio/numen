package source

import (
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/cutting"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/epub"
	"github.com/jiva-studio/numen/modules/libs/core/text"
)

const bookPath = "library/book.epub"

func TestABookIsCutIntoChunksRecordedWithTheRecipeThatCutThem(t *testing.T) {
	cases := []struct {
		name  string
		sizes cutting.Sizes
	}{
		{"at the sizes configuration names none for", cutting.Sizes{}},
		{"at sizes of its own", cutting.Sizes{Large: 60, LargeOverlap: 10, Small: 20, SmallOverlap: 5}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := t.Context()
			index, shelf := newStore(), newLibrary()
			raw := bookOf(t, "A Book", words(sanskrit, 400), words(sanskrit, 400))
			shelf.hold(bookPath, domain.KindBook, raw, 1)

			extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Owing: index, Sizes: c.sizes}
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
			if source.Ref.Size != int64(len(raw)) || source.Ref.MTime != 1 {
				t.Errorf("the source was recorded as %+v, want the file as it is", source.Ref)
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

	res, err := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Owing: index}.Execute(ctx, first)
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

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Owing: index}
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

func TestTheRecipeFollowsTheCutSizesAndStalenessFollowsTheRecipe(t *testing.T) {
	ctx := t.Context()
	one := cutting.Sizes{Large: 100, LargeOverlap: 20, Small: 20, SmallOverlap: 5}
	other := one
	other.Small = 21

	if recipe(text.ReaderEPUB, one) == recipe(text.ReaderEPUB, other) {
		t.Fatal("one word of difference in a chunk size is not in the recipe")
	}

	index, shelf := newStore(), newLibrary()
	shelf.hold(bookPath, domain.KindBook, bookOf(t, "A Book", words(sanskrit, 300)), 1)

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Owing: index, Sizes: one}
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

	extract := Extract{Readers: shelves, Sources: index, Owing: index}
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

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Owing: index}
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

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Owing: index}
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
	hash := fingerprint(raw)
	if err := made.Write(ctx, text.Partial("ocr", hash), []byte(read)); err != nil {
		t.Fatal(err)
	}

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Owing: index, Derived: made}
	if _, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}
	src := index.sources[first.ID][bookPath]
	if src.TextFrom != "ocr" {
		t.Fatalf("the source names %q as the producer of its text, want ocr", src.TextFrom)
	}

	of := text.Reader{Vault: shelf, Derived: made}
	doc, err := of.Of(ctx, bookPath, src.TextFrom, src.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Text != read {
		t.Errorf("the source reads back as %q, want the pages the run has read", doc.Text)
	}

	if err := made.Write(ctx, text.Artifact("ocr", hash), []byte(whole)); err != nil {
		t.Fatal(err)
	}
	doc, err = of.Of(ctx, bookPath, src.TextFrom, src.Hash)
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
		for _, name := range text.Names("ocr", fingerprint(raw)) {
			if err := made.Write(ctx, name, []byte("What the pages say.")); err != nil {
				t.Fatal(err)
			}
		}
	}

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Owing: index, Derived: made}
	if _, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}
	if index.sources[first.ID]["library/gone.epub"].TextFrom == "" {
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

	for _, name := range text.Names("ocr", fingerprint(gone)) {
		if _, err := made.Read(ctx, name); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s is still in the store", name)
		}
	}
	for _, name := range text.Names("ocr", fingerprint(kept)) {
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
	for _, name := range text.Names("ocr", fingerprint(raw)) {
		if err := made.Write(ctx, name, []byte("What the pages say.")); err != nil {
			t.Fatal(err)
		}
	}

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Owing: index, Derived: made}
	if _, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}
	if index.sources[first.ID]["library/before.epub"].TextFrom == "" {
		t.Fatal("the book was not cut from its reading, so renaming it proves nothing")
	}

	delete(shelf.files, "library/before.epub")
	shelf.hold("library/after.epub", domain.KindBook, raw, 1)
	if _, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}

	for _, name := range text.Names("ocr", fingerprint(raw)) {
		if _, err := made.Read(ctx, name); err != nil {
			t.Errorf("renaming the book threw away %s: %v", name, err)
		}
	}
	if from := index.sources[first.ID]["library/after.epub"].TextFrom; from == "" {
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
	for _, name := range text.Names("ocr", fingerprint(raw)) {
		if err := made.Write(ctx, name, []byte("What the pages say.")); err != nil {
			t.Fatal(err)
		}
	}

	extract := Extract{Readers: vaults{first.ID: shelf}, Sources: index, Owing: index, Derived: made}
	if _, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}

	delete(shelf.files, "library/one.epub")
	if _, err := extract.Execute(ctx, first); err != nil {
		t.Fatal(err)
	}

	for _, name := range text.Names("ocr", fingerprint(raw)) {
		if _, err := made.Read(ctx, name); err != nil {
			t.Errorf("one copy going took %s with it: %v", name, err)
		}
	}
	if from := index.sources[first.ID]["shelf/two.epub"].TextFrom; from == "" {
		t.Error("the copy that stayed lost its reading")
	}
}
