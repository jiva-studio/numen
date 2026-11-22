package index

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/filetext"
)

// suppliedTexts is the text a plugin supplies for the files of a vault: a file has text once a test
// supplies it, made from the bytes the file holds then.
type suppliedTexts struct {
	readers vaultreader.Readers

	mu    sync.Mutex
	texts map[string]domain.VersionText
}

func (s *suppliedTexts) supply(t *testing.T, v domain.Vault, path, content string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(v.Path, path))
	if err != nil {
		t.Fatal(err)
	}
	hash := domain.HashContent(raw)
	version := domain.TextVersion{Path: path, PluginID: "test.reader", Key: hash[:12], Kind: "page-text", InputHash: hash, TextHash: hash}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.texts[path] = domain.NewVersionText(version, []domain.TextSegment{{Text: content, Selector: `{"page":1}`}})
}

func (s *suppliedTexts) Open(v domain.Vault) (textport.VaultTexts, error) {
	files, err := s.readers.Open(v)
	if err != nil {
		return nil, err
	}
	return vaultSuppliedTexts{texts: s, vault: v, hash: func(ctx context.Context, path string) (string, error) {
		raw, err := files.Read(ctx, path)
		return domain.HashContent(raw), err
	}}, nil
}

type vaultSuppliedTexts struct {
	texts *suppliedTexts
	vault domain.Vault
	hash  func(ctx context.Context, path string) (string, error)
}

func (t vaultSuppliedTexts) find(path string) (domain.VersionText, bool) {
	t.texts.mu.Lock()
	defer t.texts.mu.Unlock()
	doc, ok := t.texts.texts[path]
	return doc, ok
}

func (t vaultSuppliedTexts) FindLiveVersion(ctx context.Context, path, contentHash string) (domain.TextVersionID, error) {
	doc, err := t.GetLiveText(ctx, path, contentHash)
	return doc.Version().GetID(), err
}

func (t vaultSuppliedTexts) GetLiveText(_ context.Context, path, contentHash string) (domain.VersionText, error) {
	doc, ok := t.find(path)
	if !ok || doc.Version().InputHash != contentHash {
		return domain.VersionText{}, text.ErrNoText
	}
	return doc, nil
}

func (t vaultSuppliedTexts) GetVersionText(_ context.Context, path string, id domain.TextVersionID) (domain.VersionText, error) {
	doc, ok := t.find(path)
	if !ok || doc.Version().GetID() != id {
		return domain.VersionText{}, text.ErrNoText
	}
	return doc, nil
}

func (t vaultSuppliedTexts) GetContentHash(ctx context.Context, path string) (string, error) {
	return t.hash(ctx, path)
}

func (t vaultSuppliedTexts) IsTextSource(_ context.Context, path string) (bool, error) {
	_, ok := t.find(path)
	return ok, nil
}

// fileVault is a vault folder on disk indexed by the file text index over the real index.
type fileVault struct {
	db    *DB
	v     domain.Vault
	texts *suppliedTexts
	index filetext.Index
}

func openFileVault(t *testing.T) *fileVault {
	t.Helper()
	db := openDB(t)
	v := domain.Vault{ID: getFirstVault().ID, Name: "first", Path: t.TempDir()}
	texts := &suppliedTexts{readers: vaultreader.Readers{}, texts: map[string]domain.VersionText{}}
	sources := db.Sources()
	index := filetext.NewIndex(filetext.IndexDeps{
		Readers: vaultreader.Readers{}, Sources: sources, Records: sources, Backlog: sources, Lookup: sources, Texts: texts,
	})
	return &fileVault{db: db, v: v, texts: texts, index: index}
}

func (f *fileVault) write(t *testing.T, path, content string, modified time.Time) {
	t.Helper()
	full := filepath.Join(f.v.Path, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(full, modified, modified); err != nil {
		t.Fatal(err)
	}
}

func (f *fileVault) run(t *testing.T) {
	t.Helper()
	if _, err := f.index.Execute(t.Context(), f.v); err != nil {
		t.Fatal(err)
	}
}

func (f *fileVault) readContentHash(t *testing.T, path string) string {
	t.Helper()
	hashes := rowsOfText(t, f.db, `SELECT coalesce(content_hash, '') FROM sources
		WHERE vault_id = (SELECT id FROM vaults WHERE identifier = ?) AND path = ?`, f.v.ID, path)
	if len(hashes) != 1 {
		t.Fatalf("%s is held as %v", path, hashes)
	}
	return hashes[0]
}

func rowsOfText(t *testing.T, db *DB, query string, args ...any) []string {
	t.Helper()
	rows, err := db.read.QueryContext(t.Context(), query, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Error(err)
		}
	}()
	var out []string
	for rows.Next() {
		var one string
		if err := rows.Scan(&one); err != nil {
			t.Fatal(err)
		}
		out = append(out, one)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func getFirstTime() time.Time { return time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC) }

func TestANewPDFWithNoTextHasASourceAndAnAddedChangeAndNoChunks(t *testing.T) {
	f := openFileVault(t)
	f.write(t, "books/river.pdf", "%PDF-1.7 pages of a river", getFirstTime())

	f.run(t)

	getSourceID(t, f.db, f.v, "books/river.pdf")
	if got := getEvents(listChanges(t, f.db, f.v, 0)); !slices.Equal(got, []string{"books/river.pdf file.added"}) {
		t.Errorf("the feed holds %v, want the PDF added", got)
	}
	if chunks := chunksOf(t, f.db, f.v, "books/river.pdf"); len(chunks) != 0 {
		t.Errorf("a PDF with no text has %d chunks", len(chunks))
	}
	if hash := f.readContentHash(t, "books/river.pdf"); hash != "" {
		t.Errorf("a PDF with no text has the content hash %q", hash)
	}
}

func TestPDFTextAddedLaterIsChunked(t *testing.T) {
	f := openFileVault(t)
	f.write(t, "books/river.pdf", "%PDF-1.7 pages of a river", getFirstTime())
	f.run(t)
	head := getFeedHead(t, f.db, f.v)

	f.texts.supply(t, f.v, "books/river.pdf", "The river runs to the sea through the delta of the plain.")
	res, err := f.index.IndexOne(t.Context(), f.v, "books/river.pdf")
	if err != nil {
		t.Fatal(err)
	}

	if res.Extracted != 1 || len(chunksOf(t, f.db, f.v, "books/river.pdf")) == 0 {
		t.Errorf("the text supplied for the PDF was not cut: %+v", res)
	}
	if got := getEvents(listChanges(t, f.db, f.v, head.Cursor.Seq)); len(got) != 0 {
		t.Errorf("cutting the text supplied for the PDF stored the changes %v", got)
	}
}

func TestUnchangedFileWithNoTextIsNotRewritten(t *testing.T) {
	f := openFileVault(t)
	f.write(t, "pictures/cover.png", "pixels", getFirstTime())
	f.run(t)
	head := getFeedHead(t, f.db, f.v)

	f.run(t)

	if got := getFeedHead(t, f.db, f.v); got != head {
		t.Errorf("a second run moved the feed from %+v to %+v", head, got)
	}
}

func TestAChangeAMoveAndARemovalOfAFileWithNoTextAreInTheFeed(t *testing.T) {
	f := openFileVault(t)
	f.write(t, "pictures/cover.png", "pixels", getFirstTime())
	f.write(t, "pictures/back.png", "other pixels", getFirstTime())
	f.write(t, "pictures/spine.png", "spine pixels", getFirstTime())
	f.run(t)
	cover, back := getSourceID(t, f.db, f.v, "pictures/cover.png"), getSourceID(t, f.db, f.v, "pictures/back.png")
	head := getFeedHead(t, f.db, f.v)

	f.write(t, "pictures/cover.png", "new pixels", getFirstTime().Add(time.Hour))
	if err := os.Remove(filepath.Join(f.v.Path, "pictures/spine.png")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(f.v.Path, "covers"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(f.v.Path, "pictures/back.png"), filepath.Join(f.v.Path, "covers/back.png")); err != nil {
		t.Fatal(err)
	}
	if err := f.db.Sources().MoveSources(t.Context(), f.v.ID, "pictures/back.png", "covers/back.png"); err != nil {
		t.Fatal(err)
	}
	f.run(t)

	changes := listChanges(t, f.db, f.v, head.Cursor.Seq)
	got := getEvents(changes)
	slices.Sort(got)
	want := []string{"covers/back.png file.moved", "pictures/cover.png file.changed", "pictures/spine.png file.removed"}
	if !slices.Equal(got, want) {
		t.Errorf("the feed holds %v, want %v", got, want)
	}
	for _, one := range changes {
		if one.Event == "file.changed" && one.SourceID != cover || one.Event == "file.moved" && one.SourceID != back {
			t.Errorf("%s is of source %d", one.Event, one.SourceID)
		}
	}
}
