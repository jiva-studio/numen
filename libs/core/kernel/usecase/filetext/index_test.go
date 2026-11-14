package filetext

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfake"
	"github.com/jiva-studio/numen/libs/note/domain"
)

const bookPath = "library/book.epub"

func TestABookIsSplitIntoChunksRecordedWithTheRecipeThatSplitThem(t *testing.T) {
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
			index, shelf := indexfake.NewStore(), indexfake.NewLibrary()
			raw := indexfake.CreateBook(t, "A Book", indexfake.GetWords(indexfake.GetSanskrit(), 400), indexfake.GetWords(indexfake.GetSanskrit(), 400))
			shelf.Put(bookPath, domain.KindFile, raw, 1)

			extract := newIndex(indexfake.Vaults{indexfake.GetFirst().ID: shelf}, index)
			extract.sizes = c.sizes
			res, err := extract.Execute(ctx, indexfake.GetFirst())
			if err != nil {
				t.Fatal(err)
			}
			if res.Seen != 1 || res.Recorded != 1 || res.Extracted != 1 || res.Unreadable != 0 {
				t.Errorf("the run reports %+v, want one book found, recorded and extracted", res)
			}
			if res.Chunks != len(index.Chunks) {
				t.Errorf("the run counted %d chunks and wrote %d", res.Chunks, len(index.Chunks))
			}
			if index.Extractions[bookPath] != 1 {
				t.Errorf("the source was written %d times, want one write", index.Extractions[bookPath])
			}

			source := index.Sources[indexfake.GetFirst().ID][bookPath]
			if want := chunking.GetTextVersionRecipe(extract.sizes.Resolve()); source.Recipe != want {
				t.Errorf("recipe = %q, want %q", source.Recipe, want)
			}
			if source.ContentHash == "" {
				t.Error("nothing addresses the content the chunks were cut from")
			}
			if source.FileStat.Size != int64(len(raw)) ||
				!source.FileStat.ModTime.Equal(time.Unix(0, 1)) {
				t.Errorf("the source was recorded as %+v, want the file as it is", source.FileStat)
			}

			// A chunk keeps a place in the extracted text and not the text
			book, err := indexfake.ReadBook(raw)
			if err != nil {
				t.Fatal(err)
			}
			large, small, located := 0, 0, 0
			for _, c := range index.Chunks {
				if c.Start < 0 || c.Start+c.Length > len(book.Text) {
					t.Fatalf("chunk %d runs outside the book's text", c.ID)
				}
				if got := book.Text[c.Start : c.Start+c.Length]; got != c.Text {
					t.Errorf("chunk %d says %q and its place holds %q", c.ID, c.Text, got)
				}
				if c.Parent == 0 {
					large++
				} else {
					small++
				}
				if strings.Contains(c.Location, "Part 2") {
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
	index, shelf := indexfake.NewStore(), indexfake.NewLibrary()

	// The unreadable one sorts between the two books, so a run that stops at it
	// leaves the second uncut.
	shelf.Put("library/a.epub", domain.KindFile, indexfake.CreateBook(t, "A", indexfake.GetWords(indexfake.GetSanskrit(), 200)), 1)
	shelf.Put("library/broken.epub", domain.KindFile, []byte("this is not an archive at all"), 1)
	shelf.Put("library/z.epub", domain.KindFile, indexfake.CreateBook(t, "Z", indexfake.GetWords(indexfake.GetSanskrit(), 200)), 1)
	shelf.Put("notes/one.md", domain.KindNote, []byte("# One\n\nA note.\n"), 1)

	res, err := newIndex(indexfake.Vaults{indexfake.GetFirst().ID: shelf}, index).Execute(ctx, indexfake.GetFirst())
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
		if !index.IsCut(indexfake.GetFirst().ID, path) {
			t.Errorf("%s was not cut", path)
		}
	}
	if index.IsCut(indexfake.GetFirst().ID, "library/broken.epub") {
		t.Error("something that is not a book was cut into chunks")
	}
	if got := shelf.Reads["library/broken.epub"]; got != 1 {
		t.Errorf("the unreadable book was read %d times, want one attempt in a run", got)
	}
	broken, recognized := index.Sources[indexfake.GetFirst().ID]["library/broken.epub"]
	if !recognized {
		t.Error("the file the vault holds is not a source the index knows about")
	}
	if broken.Recipe != "" {
		t.Errorf("a file nothing could read claims to have been cut by %q", broken.Recipe)
	}
	if index.IsCut(indexfake.GetFirst().ID, "notes/one.md") {
		t.Error("a note was cut by the pass over books")
	}
}

func TestWhatHasNotChangedIsNotOpenedAgain(t *testing.T) {
	ctx := t.Context()
	index, shelf := indexfake.NewStore(), indexfake.NewLibrary()
	shelf.Put(bookPath, domain.KindFile, indexfake.CreateBook(t, "A Book", indexfake.GetWords(indexfake.GetSanskrit(), 300)), 1)

	extract := newIndex(indexfake.Vaults{indexfake.GetFirst().ID: shelf}, index)
	if _, err := extract.Execute(ctx, indexfake.GetFirst()); err != nil {
		t.Fatal(err)
	}
	reads := shelf.Reads[bookPath]

	res, err := extract.Execute(ctx, indexfake.GetFirst())
	if err != nil {
		t.Fatal(err)
	}
	if res.Unchanged != 1 || res.Recorded != 0 || res.Extracted != 0 {
		t.Errorf("the second run reports %+v, want the book skipped on its fingerprint", res)
	}
	if shelf.Reads[bookPath] != reads {
		t.Errorf("the file was read %d times over two runs, want %d", shelf.Reads[bookPath], reads)
	}
	if index.Extractions[bookPath] != 1 {
		t.Errorf("the source was written %d times, want the one write", index.Extractions[bookPath])
	}

	shelf.Put(bookPath, domain.KindFile, indexfake.CreateBook(t, "A Book", indexfake.GetWords(indexfake.GetSanskrit(), 400)), 2)
	res, err = extract.Execute(ctx, indexfake.GetFirst())
	if err != nil {
		t.Fatal(err)
	}
	if res.Recorded != 1 || res.Extracted != 1 {
		t.Errorf("the run after the file changed reports %+v, want it read again", res)
	}
	if index.Extractions[bookPath] != 2 {
		t.Errorf("the source was written %d times, want a second write", index.Extractions[bookPath])
	}
}

// An overlap equal to its size is reduced before splitting, which keeps the
// tiling advancing. A recipe recording the configured number would record
// sizes that were never used, and the source would be re-read on every run.
func TestTheRecipeNamesTheSizesTheSplitCachedTo(t *testing.T) {
	request := chunking.Sizes{Large: 100, LargeOverlap: 100, Small: 20, SmallOverlap: 20}
	resolved := request.Resolve()

	if resolved.LargeOverlap != 99 || resolved.SmallOverlap != 19 {
		t.Fatalf("the cut keeps to %+v", resolved)
	}
	if got := request.Resolve(); got != resolved {
		t.Errorf("the recipe names %+v, want %+v", got, resolved)
	}
}

func TestIndex_RecipeFollowsSizesAndDrivesStaleness(t *testing.T) {
	ctx := t.Context()
	one := chunking.Sizes{Large: 100, LargeOverlap: 20, Small: 20, SmallOverlap: 5}
	other := one
	other.Small = 21

	if chunking.GetTextVersionRecipe(one) == chunking.GetTextVersionRecipe(other) {
		t.Fatal("one word of difference in a chunk size is not in the recipe")
	}

	index, shelf := indexfake.NewStore(), indexfake.NewLibrary()
	shelf.Put(bookPath, domain.KindFile, indexfake.CreateBook(t, "A Book", indexfake.GetWords(indexfake.GetSanskrit(), 300)), 1)

	extract := newIndex(indexfake.Vaults{indexfake.GetFirst().ID: shelf}, index)
	extract.sizes = one
	if _, err := extract.Execute(ctx, indexfake.GetFirst()); err != nil {
		t.Fatal(err)
	}

	stale, err := index.GetSourcesWithOtherRecipe(ctx, indexfake.GetFirst().ID, domain.KindFile, []string{chunking.GetTextVersionRecipe(other.Resolve())}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 1 || stale[0] != bookPath {
		t.Errorf("%v were cut by another recipe, want the one book", stale)
	}

	extract.sizes = other
	res, err := extract.Execute(ctx, indexfake.GetFirst())
	if err != nil {
		t.Fatal(err)
	}
	if res.Extracted != 1 || res.Recorded != 0 {
		t.Errorf("the run at other sizes reports %+v, want the book cut again and nothing recorded", res)
	}
	if index.Extractions[bookPath] != 2 {
		t.Errorf("the source was written %d times, want the cut at each of the two sizes", index.Extractions[bookPath])
	}
	if want := chunking.GetTextVersionRecipe(extract.sizes.Resolve()); index.Sources[indexfake.GetFirst().ID][bookPath].Recipe != want {
		t.Errorf("recipe = %q, want %q", index.Sources[indexfake.GetFirst().ID][bookPath].Recipe, want)
	}

	if _, err := extract.Execute(ctx, indexfake.GetFirst()); err != nil {
		t.Fatal(err)
	}
	if index.Extractions[bookPath] != 2 {
		t.Errorf("a run at the sizes already used wrote the source again, %d writes in all", index.Extractions[bookPath])
	}
}

func TestExtractionStaysInsideItsVault(t *testing.T) {
	ctx := t.Context()
	index := indexfake.NewStore()

	shelves := indexfake.Vaults{indexfake.GetFirst().ID: indexfake.NewLibrary(), indexfake.GetSecond().ID: indexfake.NewLibrary()}
	shelves[indexfake.GetFirst().ID].Put("library/sanskrit.epub", domain.KindFile,
		indexfake.CreateBook(t, "Sanskrit", indexfake.GetWords(indexfake.GetSanskrit(), 300), indexfake.GetWords(indexfake.GetSanskrit(), 300)), 1)
	shelves[indexfake.GetSecond().ID].Put("library/latin.epub", domain.KindFile,
		indexfake.CreateBook(t, "Latin", indexfake.GetWords(indexfake.GetLatin(), 300), indexfake.GetWords(indexfake.GetLatin(), 300)), 1)

	extract := newIndex(shelves, index)
	for _, v := range []domain.Vault{indexfake.GetFirst(), indexfake.GetSecond()} {
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
		{indexfake.GetFirst(), "library/sanskrit.epub", indexfake.GetSanskrit()[0], indexfake.GetLatin()[0]},
		{indexfake.GetSecond(), "library/latin.epub", indexfake.GetLatin()[0], indexfake.GetSanskrit()[0]},
	}
	for _, c := range cases {
		t.Run(c.vault.Name, func(t *testing.T) {
			chunks := index.GetSmallChunks(c.vault.ID)
			if len(chunks) == 0 {
				t.Fatalf("%s holds no chunk that carries a vector", c.vault.Name)
			}
			for _, chunk := range chunks {
				if chunk.Path != c.path {
					t.Fatalf("%s holds a chunk of %s", c.vault.Name, chunk.Path)
				}
				if !strings.Contains(chunk.Text, c.holds) {
					t.Fatalf("a chunk of %s is not written in its words: %q", c.vault.Name, chunk.Text)
				}
				if strings.Contains(chunk.Text, c.absent) {
					t.Fatalf("a chunk of %s holds the other vault's words: %q", c.vault.Name, chunk.Text)
				}
			}
			recognized, err := index.GetFileStats(ctx, c.vault.ID, domain.KindFile)
			if err != nil {
				t.Fatal(err)
			}
			if len(recognized) != 1 || recognized[c.path].Path != c.path {
				t.Errorf("%s knows about %v, want its own book alone", c.vault.Name, recognized)
			}
		})
	}
}

// Removal takes a book's source row and every chunk cut from it out of the
// index. The text of a passage is read from its file, so a passage naming a file
// that is gone can answer nothing.
func TestABookTheVaultNoLongerContainsIsTakenOut(t *testing.T) {
	ctx := t.Context()
	index, shelf := indexfake.NewStore(), indexfake.NewLibrary()

	shelf.Put("library/kept.epub", domain.KindFile, indexfake.CreateBook(t, "Kept", indexfake.GetWords(indexfake.GetSanskrit(), 200)), 1)
	shelf.Put("library/gone.epub", domain.KindFile, indexfake.CreateBook(t, "Gone", indexfake.GetWords(indexfake.GetLatin(), 200)), 1)

	extract := newIndex(indexfake.Vaults{indexfake.GetFirst().ID: shelf}, index)
	if _, err := extract.Execute(ctx, indexfake.GetFirst()); err != nil {
		t.Fatal(err)
	}
	if !index.IsCut(indexfake.GetFirst().ID, "library/gone.epub") {
		t.Fatal("the book was not cut, so its removal proves nothing")
	}

	delete(shelf.Files, "library/gone.epub")

	res, err := extract.Execute(ctx, indexfake.GetFirst())
	if err != nil {
		t.Fatal(err)
	}
	if res.Removed != 1 {
		t.Errorf("removed = %d, want the one book the vault no longer holds", res.Removed)
	}
	if _, recognized := index.Sources[indexfake.GetFirst().ID]["library/gone.epub"]; recognized {
		t.Error("a book the vault does not hold is still a source the index knows about")
	}
	if index.IsCut(indexfake.GetFirst().ID, "library/gone.epub") {
		t.Error("the chunks of a book that is gone are still in the index")
	}

	// The one that stayed is untouched: removal names paths.
	if _, recognizedVal := index.Sources[indexfake.GetFirst().ID]["library/kept.epub"]; !recognizedVal {
		t.Error("the book that is still there was taken out as well")
	}
	if !index.IsCut(indexfake.GetFirst().ID, "library/kept.epub") {
		t.Error("the chunks of the book that is still there were taken out as well")
	}
}

// A file whose content changed while its size and modification time did not is
// skipped: the index compares only those three. An archive restored by `unzip`
// and a tree synced by `rsync -tc` both produce this, and the chunks then
// describe text the file does not contain. Rebuilding the index reads every file.
func TestRebuildingTheIndexReadsEveryFile(t *testing.T) {
	ctx := t.Context()
	index, shelf := indexfake.NewStore(), indexfake.NewLibrary()

	was := indexfake.GetWords(indexfake.GetLatin(), 200)
	// Other words, letter for letter, so the size is the same one.
	now := strings.Map(func(r rune) rune {
		if r == 'a' {
			return 'e'
		}
		return r
	}, was)
	before, after := indexfake.CreateBook(t, "A", was), indexfake.CreateBook(t, "A", now)
	if len(before) != len(after) || string(before) == string(after) {
		t.Fatalf("the two books are %d and %d bytes, and this asks about equal ones",
			len(before), len(after))
	}

	shelf.Put("library/A.epub", domain.KindFile, before, 1)

	extract := newIndex(indexfake.Vaults{indexfake.GetFirst().ID: shelf}, index)
	if res, err := extract.Execute(ctx, indexfake.GetFirst()); err != nil {
		t.Fatal(err)
	} else if res.Extracted != 1 {
		t.Fatalf("read %d books", res.Extracted)
	}

	shelf.Put("library/A.epub", domain.KindFile, after, 1)

	if res, err := extract.Execute(ctx, indexfake.GetFirst()); err != nil {
		t.Fatal(err)
	} else if res.Unchanged != 1 || res.Extracted != 0 {
		t.Errorf("the fingerprint is the path, the size and the time: %+v", res)
	}

	extract.shouldRebuildIndex = true
	if res, err := extract.Execute(ctx, indexfake.GetFirst()); err != nil {
		t.Fatal(err)
	} else if res.Extracted != 1 {
		t.Errorf("rebuilding read %d books: %+v", res.Extracted, res)
	}
}

const talkPath = "talks/a lecture.mp3"

func timedParts() []domain.TextSegment {
	return []domain.TextSegment{
		{Text: indexfake.GetWords(indexfake.GetSanskrit(), 200) + "\n", Selector: `{"t":[1.5,4]}`},
		{Text: indexfake.GetWords(indexfake.GetLatin(), 200) + "\n", Selector: `{"t":[5025,5030]}`},
	}
}

// A recording with no text version is a source identified by its name, with
// nothing to split into chunks.
func TestARecordingWithNoVersionIsASourceWithNoChunks(t *testing.T) {
	ctx := t.Context()
	index, shelf := indexfake.NewStore(), indexfake.NewLibrary()
	shelf.Put(talkPath, domain.KindFile, []byte("ID3 and then the samples"), 1)

	extract, _ := newIndexWithTexts(indexfake.Vaults{indexfake.GetFirst().ID: shelf}, index)
	res, err := extract.Execute(ctx, indexfake.GetFirst())
	if err != nil {
		t.Fatal(err)
	}
	if res.Seen != 1 || res.Recorded != 1 || res.Unreadable != 1 {
		t.Errorf("the run reports %+v, want the recording found, recorded and without text", res)
	}
	if res.Chunks != 0 || res.Extracted != 0 {
		t.Errorf("the run cut %d chunks out of a recording with no text", res.Chunks)
	}
	if _, target := index.Sources[indexfake.GetFirst().ID][talkPath]; !target {
		t.Error("the vault holds a recording the index does not")
	}
}

func TestIndexSkipsAFileWithNoTextSource(t *testing.T) {
	ctx := t.Context()
	index, shelf := indexfake.NewStore(), indexfake.NewLibrary()
	shelf.Put("pictures/cover.sample", domain.KindFile, []byte("pixels"), 1)

	run, texts := newIndexWithTexts(indexfake.Vaults{indexfake.GetFirst().ID: shelf}, index)
	texts.leaveOut("pictures/cover.sample")
	res, err := run.Execute(ctx, indexfake.GetFirst())
	if err != nil {
		t.Fatal(err)
	}
	if res.Seen != 0 || res.Recorded != 0 {
		t.Errorf("the run reports %+v, want the file left out", res)
	}
	if _, isIndexed := index.Sources[indexfake.GetFirst().ID]["pictures/cover.sample"]; isIndexed {
		t.Error("the index holds a file that has no text")
	}
}

// The live version of a recording is its text.
func TestARecordingIsSplitFromItsLiveVersion(t *testing.T) {
	ctx := t.Context()
	index, shelf := indexfake.NewStore(), indexfake.NewLibrary()
	shelf.Put(talkPath, domain.KindFile, []byte("ID3 and then the samples"), 1)

	extract, texts := newIndexWithTexts(indexfake.Vaults{indexfake.GetFirst().ID: shelf}, index)
	transcript := texts.put(indexfake.GetFirst(), talkPath, "test.transcriber", "transcript", timedParts())
	res, err := extract.Execute(ctx, indexfake.GetFirst())
	if err != nil {
		t.Fatal(err)
	}
	if res.Extracted != 1 || res.Chunks == 0 {
		t.Fatalf("the run reports %+v, want the recording cut into chunks", res)
	}
	src := index.Sources[indexfake.GetFirst().ID][talkPath]
	if src.Producer != string(transcript.GetID()) {
		t.Errorf("the source names %q as the version it was cut from, want %q", src.Producer, transcript.GetID())
	}
	if want := chunking.GetTextVersionRecipe(extract.sizes.Resolve()); src.Recipe != want {
		t.Errorf("recipe = %q, want %q", src.Recipe, want)
	}
}

// A version that stops being the live one leaves its chunks pending: the next run
// splits the file from the current live version, or stores no chunks for it.
func TestSourceExtractedFromOutdatedVersionIsExtractedAgain(t *testing.T) {
	ctx := t.Context()
	index, shelf := indexfake.NewStore(), indexfake.NewLibrary()
	shelf.Put(talkPath, domain.KindFile, []byte("ID3 and then the samples"), 1)

	extract, texts := newIndexWithTexts(indexfake.Vaults{indexfake.GetFirst().ID: shelf}, index)
	texts.put(indexfake.GetFirst(), talkPath, "test.transcriber", "transcript", timedParts())
	if _, err := extract.Execute(ctx, indexfake.GetFirst()); err != nil {
		t.Fatal(err)
	}
	if !index.IsCut(indexfake.GetFirst().ID, talkPath) {
		t.Fatal("the recording was not cut, so dropping its version proves nothing")
	}

	texts.drop(talkPath)
	res, err := extract.Execute(ctx, indexfake.GetFirst())
	if err != nil {
		t.Fatal(err)
	}
	if res.Outdated != 1 || res.Unreadable != 1 {
		t.Errorf("the run reports %+v, want the source outdated and left without text", res)
	}
	if index.IsCut(indexfake.GetFirst().ID, talkPath) {
		t.Error("the chunks of a version that is not live are still in the index")
	}
	if got := index.Sources[indexfake.GetFirst().ID][talkPath].Producer; got != "" {
		t.Errorf("the source still names %q as its version", got)
	}
}

// ExtractOne is the cut the index makes when a file's versions change.
func TestExtractOneUsesTheLiveVersion(t *testing.T) {
	ctx := t.Context()
	index, shelf := indexfake.NewStore(), indexfake.NewLibrary()
	shelf.Put(talkPath, domain.KindFile, []byte("ID3 and then the samples"), 1)

	extract, texts := newIndexWithTexts(indexfake.Vaults{indexfake.GetFirst().ID: shelf}, index)
	if _, err := extract.Execute(ctx, indexfake.GetFirst()); err != nil {
		t.Fatal(err)
	}
	transcript := texts.put(indexfake.GetFirst(), talkPath, "test.transcriber", "transcript", timedParts())
	if _, err := extract.IndexOne(ctx, indexfake.GetFirst(), talkPath); err != nil {
		t.Fatal(err)
	}
	if got := index.Sources[indexfake.GetFirst().ID][talkPath].Producer; got != string(transcript.GetID()) {
		t.Fatalf("the source names %q, want %q", got, transcript.GetID())
	}
	if !index.IsCut(indexfake.GetFirst().ID, talkPath) {
		t.Fatal("the new version was not cut")
	}
}

// A provider that cannot respond is one failed source: it is reported to the
// error handler and the run continues.
func TestATextThatCannotBeReadIsReportedAndTheRunGoesOn(t *testing.T) {
	ctx := t.Context()
	index, shelf := indexfake.NewStore(), indexfake.NewLibrary()
	shelf.Put(bookPath, domain.KindFile, indexfake.CreateBook(t, "A Book", indexfake.GetWords(indexfake.GetSanskrit(), 200)), 1)

	extract, texts := newIndexWithTexts(indexfake.Vaults{indexfake.GetFirst().ID: shelf}, index)
	texts.failure = errors.New("the plugin is not running")
	var errs []error
	extract.errorHandler = func(err error) { errs = append(errs, err) }
	res, err := extract.Execute(ctx, indexfake.GetFirst())
	if err != nil {
		t.Fatalf("a provider that failed ended the run: %v", err)
	}
	if res.Unreadable != 1 || len(errs) != 1 {
		t.Fatalf("the run reports %+v and the handler got %v, want one unreadable source reported", res, errs)
	}
}

func TestTheRecipeNamesTheVersionReaderAndTheSizes(t *testing.T) {
	sizes := chunking.Sizes{}.Resolve()
	if want := "text-version-1/large=200+40/small=50+10/limit=1000"; chunking.GetTextVersionRecipe(sizes) != want {
		t.Errorf("a source is cut by %q, want %q", chunking.GetTextVersionRecipe(sizes), want)
	}
}

const linkPath = "links/tidal mill.md"

// A note whose chunks were cut from a text version is read from its own file
// again once the version is deleted, and a note no version was cut for is left
// to the notes pass.
func TestANoteWhoseVersionIsDeletedIsReadAsANoteAgain(t *testing.T) {
	ctx := t.Context()
	index, shelf := indexfake.NewStore(), indexfake.NewLibrary()
	shelf.Put(linkPath, domain.KindNote, []byte("---\nurl: http://tidal-mills.test\n---\n"), 1)
	shelf.Put("notes/plain.md", domain.KindNote, []byte("# Plain\n\nA note.\n"), 1)

	extract, texts := newIndexWithTexts(indexfake.Vaults{indexfake.GetFirst().ID: shelf}, index)
	var reread []string
	extract.readNote = func(_ context.Context, _ domain.Vault, path string) error {
		reread = append(reread, path)
		return nil
	}
	texts.put(indexfake.GetFirst(), linkPath, "test.importer", "article", timedParts())
	if _, err := extract.IndexOne(ctx, indexfake.GetFirst(), linkPath); err != nil {
		t.Fatal(err)
	}
	if !index.IsCut(indexfake.GetFirst().ID, linkPath) {
		t.Fatal("the link note was not cut from its version, so deleting it proves nothing")
	}

	texts.drop(linkPath)
	if _, err := extract.IndexOne(ctx, indexfake.GetFirst(), linkPath); err != nil {
		t.Fatal(err)
	}
	if _, err := extract.IndexOne(ctx, indexfake.GetFirst(), "notes/plain.md"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(reread, []string{linkPath}) {
		t.Errorf("the notes read again are %v, want only %s", reread, linkPath)
	}
}

func TestExecuteIndexesNoteTextVersion(t *testing.T) {
	ctx := t.Context()
	index, shelf := indexfake.NewStore(), indexfake.NewLibrary()
	shelf.Put(linkPath, domain.KindNote, []byte("---\nurl: http://tidal-mills.test\n---\n"), 1)
	shelf.Put("notes/plain.md", domain.KindNote, []byte("# Plain\n\nA note.\n"), 1)

	extract, texts := newIndexWithTexts(indexfake.Vaults{indexfake.GetFirst().ID: shelf}, index)
	var reread []string
	extract.readNote = func(_ context.Context, _ domain.Vault, path string) error {
		reread = append(reread, path)
		return nil
	}
	article := texts.put(indexfake.GetFirst(), linkPath, "test.importer", "article", timedParts())

	res, err := extract.Execute(ctx, indexfake.GetFirst())
	if err != nil {
		t.Fatal(err)
	}
	if res.Seen != 1 || res.Recorded != 1 || res.Extracted != 1 {
		t.Fatalf("first run reports %+v, want only the link note seen, recorded and extracted", res)
	}
	if got := index.Sources[indexfake.GetFirst().ID][linkPath].Producer; got != string(article.GetID()) {
		t.Errorf("producer = %q, want %q", got, article.GetID())
	}

	res, err = extract.Execute(ctx, indexfake.GetFirst())
	if err != nil {
		t.Fatal(err)
	}
	if res.Seen != 1 || res.Unchanged != 1 || res.Extracted != 0 {
		t.Errorf("second run reports %+v, want the link note unchanged", res)
	}

	texts.drop(linkPath)
	res, err = extract.Execute(ctx, indexfake.GetFirst())
	if err != nil {
		t.Fatal(err)
	}
	if res.Outdated != 1 || !slices.Equal(reread, []string{linkPath}) {
		t.Errorf("run after dropping the version reports %+v and reread %v", res, reread)
	}
}
