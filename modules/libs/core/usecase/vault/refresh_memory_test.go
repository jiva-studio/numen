package vault_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"testing"
	"time"

	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/modules/libs/core/port/vaultport"
)

var (
	errMemory = errors.New("the memory vault failed")
	memBase   = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	memVault  = domain.Vault{ID: "memory", Name: "memory", Path: "/memory"}
)

// memFile is one path of a vault held in memory: what a stat answers, what a
// read answers, and the failures either gives instead.
type memFile struct {
	fp      domain.Fingerprint
	content []byte
	statErr error
	readErr error
}

type sourceRemoval struct {
	kind  domain.SourceKind
	paths []string
}

// memWorld is a vault, the index rows of its sources and the clock, all in
// memory. A note saved into it is what the index answers for that path
// afterwards.
type memWorld struct {
	files map[string]*memFile
	item  map[string][]domain.Fingerprint
	clock func() time.Time

	openErr     error
	registerErr error
	queriesErr  error
	removeErr   error
	sourcesErr  error
	// saveFailures is how many saves fail before the rest succeed.
	saveFailures int
	// afterRead runs after every successful read.
	afterRead func()

	opens          int
	registers      int
	saves          [][]string
	removedNotes   [][]string
	removedSources []sourceRemoval
}

func newMemWorld() *memWorld {
	w := &memWorld{files: map[string]*memFile{}, item: map[string][]domain.Fingerprint{}}
	w.clock = func() time.Time { return memBase }
	return w
}

func (w *memWorld) setNow(pos time.Time) { w.clock = func() time.Time { return pos } }

func (w *memWorld) addNote(path, content string, mtime time.Time) *memFile {
	f := &memFile{
		fp:      domain.Fingerprint{Path: path, Kind: domain.KindNote, Size: int64(len(content)), ModTime: mtime},
		content: []byte(content),
	}
	w.files[path] = f
	return f
}

// index makes the index hold the file as it is.
func (w *memWorld) index(path string) {
	w.item[path] = []domain.Fingerprint{w.files[path].fp}
}

func (w *memWorld) refresh() vaults.Refresh {
	return vaults.NewRefresh(
		memReaders{w}, memRows{w}, memNotes{w}, memQueries{w: w}, memSources{w: w},
		func() time.Time { return w.clock() },
	)
}

func (w *memWorld) savedPaths() []string {
	var out []string
	for _, group := range w.saves {
		out = append(out, group...)
	}
	return out
}

type memReaders struct{ w *memWorld }

func (r memReaders) Open(domain.Vault) (vaultport.VaultReader, error) {
	r.w.opens++
	if r.w.openErr != nil {
		return nil, r.w.openErr
	}
	return memReader{w: r.w}, nil
}

type memReader struct {
	vaultport.VaultReader
	w *memWorld
}

func (r memReader) GetFingerprint(_ context.Context, path string) (domain.Fingerprint, error) {
	f, ok := r.w.files[path]
	if !ok {
		return domain.Fingerprint{}, fs.ErrNotExist
	}
	if f.statErr != nil {
		return domain.Fingerprint{}, f.statErr
	}
	return f.fp, nil
}

func (r memReader) Read(_ context.Context, path string) ([]byte, error) {
	f, ok := r.w.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	if f.readErr != nil {
		return nil, f.readErr
	}
	if r.w.afterRead != nil {
		r.w.afterRead()
	}
	return append([]byte(nil), f.content...), nil
}

type memRows struct{ w *memWorld }

func (r memRows) Register(context.Context, domain.VaultID) error {
	r.w.registers++
	return r.w.registerErr
}

func (r memRows) Unregister(context.Context, domain.VaultID) error { return nil }

type memNotes struct{ w *memWorld }

func (n memNotes) Save(_ context.Context, _ domain.VaultID, notes []domain.NoteChunks) error {
	if n.w.saveFailures > 0 {
		n.w.saveFailures--
		return errMemory
	}
	var paths []string
	for _, note := range notes {
		paths = append(paths, note.Note.Fingerprint.Path)
		n.w.item[note.Note.Fingerprint.Path] = []domain.Fingerprint{note.Note.Fingerprint}
	}
	n.w.saves = append(n.w.saves, paths)
	return nil
}

func (n memNotes) Remove(_ context.Context, _ domain.VaultID, paths []string) error {
	if n.w.removeErr != nil {
		return n.w.removeErr
	}
	n.w.removedNotes = append(n.w.removedNotes, append([]string(nil), paths...))
	return nil
}

type memQueries struct {
	sourceport.SourceQueries
	w *memWorld
}

func (q memQueries) GetSourcesUnder(_ context.Context, _ domain.VaultID, path string) ([]domain.Fingerprint, error) {
	if q.w.queriesErr != nil {
		return nil, q.w.queriesErr
	}
	return q.w.item[path], nil
}

type memSources struct {
	sourceport.SourceRepository
	w *memWorld
}

func (s memSources) RemoveSources(_ context.Context, _ domain.VaultID, kind domain.SourceKind, paths []string) error {
	if s.w.sourcesErr != nil {
		return s.w.sourcesErr
	}
	s.w.removedSources = append(s.w.removedSources, sourceRemoval{kind: kind, paths: paths})
	return nil
}

func execute(t *testing.T, refresh vaults.Refresh, paths ...string) vaults.RefreshResult {
	t.Helper()
	res, err := refresh.Execute(t.Context(), memVault, paths)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestARefreshOfNoPathsOpensNothing(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	w.openErr = errMemory

	res, err := w.refresh().Execute(t.Context(), memVault, nil)
	if err != nil {
		t.Fatalf("no paths were named, yet: %v", err)
	}
	if len(res.GetChangedPaths()) != 0 {
		t.Errorf("changed %v", res.GetChangedPaths())
	}
	if w.opens != 0 || w.registers != 0 {
		t.Errorf("opened %d times and registered %d times", w.opens, w.registers)
	}
}

func TestARefreshFailsWhenTheVaultCannotBeOpened(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	w.openErr = errMemory
	w.addNote("Note.md", "entropy", memBase.Add(-time.Hour))

	_, err := w.refresh().Execute(t.Context(), memVault, []string{"Note.md"})
	if !errors.Is(err, errMemory) {
		t.Fatalf("got %v", err)
	}
	if w.registers != 0 {
		t.Errorf("the vault was registered %d times", w.registers)
	}
}

func TestARefreshFailsWhenTheVaultCannotBeRegistered(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	w.registerErr = errMemory
	w.addNote("Note.md", "entropy", memBase.Add(-time.Hour))

	_, err := w.refresh().Execute(t.Context(), memVault, []string{"Note.md"})
	if !errors.Is(err, errMemory) {
		t.Fatalf("got %v", err)
	}
	if len(w.saves) != 0 {
		t.Errorf("saved %v", w.saves)
	}
}

func TestACancelledRefreshSavesNothing(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	w.addNote("Note.md", "entropy", memBase.Add(-time.Hour))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := w.refresh().Execute(ctx, memVault, []string{"Note.md"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if len(w.saves) != 0 {
		t.Errorf("saved %v", w.saves)
	}
}

func TestEveryNamedPathIsSortedIntoItsOutcome(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	old := memBase.Add(-time.Hour)
	w.addNote("Settled.md", "settled", old)
	w.index("Settled.md")
	w.files["Gone.md"] = &memFile{statErr: fs.ErrNotExist}
	w.files["Foreign.png"] = &memFile{statErr: vaultport.ErrNotANote}
	w.files["Locked.md"] = &memFile{statErr: fs.ErrPermission}
	w.addNote("Vanish.md", "vanish", old).readErr = fs.ErrNotExist
	w.addNote("Denied.md", "denied", old).readErr = fs.ErrPermission
	w.files["Book.epub"] = &memFile{fp: domain.Fingerprint{Path: "Book.epub", Kind: domain.KindBook, Size: 3, ModTime: old}}
	w.addNote("Edited.md", "edited", old)

	res := execute(t, w.refresh(),
		"Settled.md", "Gone.md", "Foreign.png", "Locked.md", "Vanish.md", "Denied.md", "Book.epub", "Edited.md")

	want := map[string][]string{
		"indexed":    {"Settled.md", "Edited.md"},
		"removed":    {"Gone.md"},
		"left alone": {"Foreign.png"},
		"unreadable": {"Locked.md", "Denied.md"},
		"assets":     {"Book.epub"},
	}
	got := map[string][]string{
		"indexed":    res.Indexed,
		"removed":    res.Removed,
		"left alone": res.LeftAlone,
		"unreadable": res.Unreadable,
		"assets":     res.Assets,
	}
	for name, paths := range want {
		if fmt.Sprint(got[name]) != fmt.Sprint(paths) {
			t.Errorf("%s: %v, want %v", name, got[name], paths)
		}
	}
	if fmt.Sprint(w.savedPaths()) != "[Edited.md]" {
		t.Errorf("saved %v", w.savedPaths())
	}
	if fmt.Sprint(w.removedNotes) != "[[Gone.md Foreign.png]]" {
		t.Errorf("took %v out of the note index", w.removedNotes)
	}
}

func TestARefreshFailsWhenTheIndexCannotSaveTheLastGroup(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	w.saveFailures = 1
	w.addNote("Note.md", "entropy", memBase.Add(-time.Hour))

	res, err := w.refresh().Execute(t.Context(), memVault, []string{"Note.md"})
	if !errors.Is(err, errMemory) {
		t.Fatalf("got %v", err)
	}
	if len(res.Indexed) == 0 {
		t.Errorf("the result lost what was done before the failure")
	}
}

func TestARefreshFailsWhenTheIndexCannotSaveAFullGroup(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	w.saveFailures = 1
	paths := make([]string, 0, 1200)
	for i := range 1200 {
		path := fmt.Sprintf("note-%04d.md", i)
		w.addNote(path, "entropy", memBase.Add(-time.Hour))
		paths = append(paths, path)
	}

	res, err := w.refresh().Execute(t.Context(), memVault, paths)
	if !errors.Is(err, errMemory) {
		t.Fatalf("got %v", err)
	}
	if len(res.Indexed) == 0 {
		t.Errorf("the result lost what was done before the failure")
	}
}

func TestARefreshFailsWhenTheIndexCannotRemoveANote(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	w.removeErr = errMemory
	w.files["Gone.md"] = &memFile{statErr: fs.ErrNotExist}

	res, err := w.refresh().Execute(t.Context(), memVault, []string{"Gone.md"})
	if !errors.Is(err, errMemory) {
		t.Fatalf("got %v", err)
	}
	if len(res.Removed) == 0 {
		t.Errorf("the result lost what was done before the failure")
	}
}

func TestARefreshFailsWhenTheSourcesBelowARemovedPathCannotBeListed(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	w.queriesErr = errMemory
	w.files["Gone.md"] = &memFile{statErr: fs.ErrNotExist}

	res, err := w.refresh().Execute(t.Context(), memVault, []string{"Gone.md"})
	if !errors.Is(err, errMemory) {
		t.Fatalf("got %v", err)
	}
	if len(res.Removed) == 0 {
		t.Errorf("the result lost what was done before the failure")
	}
}

func TestARefreshFailsWhenASourceCannotBeRemoved(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	w.sourcesErr = errMemory
	w.files["Gone"] = &memFile{statErr: fs.ErrNotExist}
	w.item["Gone"] = []domain.Fingerprint{{Path: "Gone/b.epub", Kind: domain.KindBook}}

	_, err := w.refresh().Execute(t.Context(), memVault, []string{"Gone"})
	if !errors.Is(err, errMemory) {
		t.Fatalf("got %v", err)
	}
}

func TestAFolderThatWentConsumesItsBooksAndRecordingsOutOfTheIndexByKind(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	w.files["Gone"] = &memFile{statErr: fs.ErrNotExist}
	w.item["Gone"] = []domain.Fingerprint{
		{Path: "Gone/a.md", Kind: domain.KindNote},
		{Path: "Gone/c.mp3", Kind: domain.KindRecording},
		{Path: "Gone/b.epub", Kind: domain.KindBook},
	}

	execute(t, w.refresh(), "Gone")

	want := []sourceRemoval{
		{kind: domain.KindBook, paths: []string{"Gone/b.epub"}},
		{kind: domain.KindRecording, paths: []string{"Gone/c.mp3"}},
	}
	if fmt.Sprint(w.removedSources) != fmt.Sprint(want) {
		t.Errorf("removed %v, want %v", w.removedSources, want)
	}
}

func TestOnlyAnIndexedNoteOfThatPathIsTakenForCurrent(t *testing.T) {
	t.Parallel()
	old := memBase.Add(-time.Hour)
	other := domain.Fingerprint{Path: "Other.md", Kind: domain.KindNote, Size: 7, ModTime: old}
	asBook := domain.Fingerprint{Path: "Note.md", Kind: domain.KindBook, Size: 7, ModTime: old}
	asNote := domain.Fingerprint{Path: "Note.md", Kind: domain.KindNote, Size: 7, ModTime: old}
	for name, tc := range map[string]struct {
		item  []domain.Fingerprint
		saved int
	}{
		"a note of another path": {item: []domain.Fingerprint{other}, saved: 1},
		"a book at the path":     {item: []domain.Fingerprint{asBook}, saved: 1},
		"the note after another": {item: []domain.Fingerprint{other, asNote}, saved: 0},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			w := newMemWorld()
			w.addNote("Note.md", "entropy", old)
			w.item["Note.md"] = tc.item

			execute(t, w.refresh(), "Note.md")

			if got := len(w.savedPaths()); got != tc.saved {
				t.Errorf("saved %d notes, want %d", got, tc.saved)
			}
		})
	}
}

func TestANoteIsRecentWhileItIsYoungerThanTheGranularity(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		age   time.Duration
		saved int
	}{
		"1.5 seconds":         {age: 1500 * time.Millisecond, saved: 1},
		"just under the step": {age: vaults.TimestampGranularity - time.Nanosecond, saved: 1},
		"exactly the step":    {age: vaults.TimestampGranularity, saved: 0},
		"2.5 seconds":         {age: 2500 * time.Millisecond, saved: 0},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			w := newMemWorld()
			w.addNote("Note.md", "entropy", memBase)
			w.index("Note.md")
			w.setNow(memBase.Add(tc.age))

			execute(t, w.refresh(), "Note.md")

			if got := len(w.savedPaths()); got != tc.saved {
				t.Errorf("saved %d notes, want %d", got, tc.saved)
			}
		})
	}
}

func TestTheLedgerContainsARecentNoteUntilItsFilePreservesTheWindow(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	w.addNote("Note.md", "entropy", memBase.Add(-time.Second))
	refresh := w.refresh()

	execute(t, refresh, "Note.md")
	if n := refresh.Ledger.Len(); n != 1 {
		t.Fatalf("the ledger holds %d notes after a recent save", n)
	}
	res := execute(t, refresh, "Note.md")

	if fmt.Sprint(res.Indexed) != "[Note.md]" || len(w.savedPaths()) != 1 {
		t.Errorf("indexed %v after %d saves", res.Indexed, len(w.savedPaths()))
	}
	if n := refresh.Ledger.Len(); n != 1 {
		t.Errorf("the ledger holds %d notes, want the one still in the window", n)
	}
}

func TestTheLedgerLetsGoOfANoteOnceItsFileHasLeftTheWindow(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	w.addNote("Note.md", "entropy", memBase.Add(-time.Second))
	refresh := w.refresh()
	execute(t, refresh, "Note.md")

	calls := 0
	w.clock = func() time.Time {
		calls++
		if calls == 1 {
			return memBase
		}
		return memBase.Add(3 * time.Second)
	}
	execute(t, refresh, "Note.md")

	if n := refresh.Ledger.Len(); n != 0 {
		t.Errorf("the ledger holds %d notes of a file that left the window", n)
	}
}

func TestANoteTheIndexChangedBelowTheLedgerIsSavedAgain(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	f := w.addNote("Note.md", "entropy", memBase.Add(-time.Second))
	refresh := w.refresh()
	execute(t, refresh, "Note.md")

	f.fp.ModTime = memBase.Add(-500 * time.Millisecond)
	w.index("Note.md")
	execute(t, refresh, "Note.md")

	if got := len(w.savedPaths()); got != 2 {
		t.Errorf("saved %d times, want 2", got)
	}
}

func TestANoteThatCannotBeReadAgainIsNotTakenForCurrent(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	f := w.addNote("Empty.md", "", memBase.Add(-time.Second))
	refresh := w.refresh()
	execute(t, refresh, "Empty.md")

	f.readErr = fs.ErrPermission
	res := execute(t, refresh, "Empty.md")

	if fmt.Sprint(res.Unreadable) != "[Empty.md]" || len(res.Indexed) != 0 {
		t.Errorf("unreadable %v, indexed %v", res.Unreadable, res.Indexed)
	}
}

func TestTheLedgerForgetsAnEntryWhoseFileLeftTheWindowWhenAnotherNoteIsSaved(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	w.addNote("A.md", "entropy", memBase)
	w.addNote("B.md", "entropy", memBase.Add(vaults.TimestampGranularity))
	w.setNow(memBase.Add(time.Second))
	refresh := w.refresh()
	execute(t, refresh, "A.md")

	w.setNow(memBase.Add(vaults.TimestampGranularity))
	execute(t, refresh, "B.md")

	if n := refresh.Ledger.Len(); n != 1 {
		t.Errorf("the ledger holds %d notes, want B only", n)
	}
}

func TestTheLedgerForgetsAnEntryWhoseFileLeftTheWindowWhenAnotherNoteIsLookedUp(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	w.addNote("A.md", "entropy", memBase)
	w.addNote("C.md", "entropy", memBase.Add(-time.Hour))
	w.index("C.md")
	w.setNow(memBase.Add(time.Second))
	refresh := w.refresh()
	execute(t, refresh, "A.md")

	w.setNow(memBase.Add(3 * time.Second))
	execute(t, refresh, "C.md")

	if n := refresh.Ledger.Len(); n != 0 {
		t.Errorf("the ledger holds %d notes of files that left the window", n)
	}
}

func TestTheLedgerForgetsANoteThatIsSavedOutsideTheWindow(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	f := w.addNote("Note.md", "entropy", memBase.Add(-time.Second))
	refresh := w.refresh()
	execute(t, refresh, "Note.md")

	f.fp.ModTime = memBase.Add(-time.Hour)
	f.fp.Size++
	execute(t, refresh, "Note.md")

	if n := refresh.Ledger.Len(); n != 0 {
		t.Errorf("the ledger holds %d notes of a file that left the window", n)
	}
}

func TestEveryRecentNoteIsRecordedWhateverTheOthersAre(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	paths := []string{"Recent.md"}
	w.addNote("Recent.md", "entropy", memBase.Add(-500*time.Millisecond))
	for i := range 8 {
		path := fmt.Sprintf("Old%d.md", i)
		w.addNote(path, "entropy", memBase.Add(-time.Hour))
		paths = append(paths, path)
	}
	refresh := w.refresh()
	refresh.Force = true

	for run := range 8 {
		refresh.Ledger = vaults.NewLedger()
		execute(t, refresh, paths...)
		if n := refresh.Ledger.Len(); n != 1 {
			t.Fatalf("run %d: the ledger holds %d notes, want the recent one", run, n)
		}
	}
}

func TestARefreshWithoutALedgerSavesARecentNoteEveryTime(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	f := w.addNote("Note.md", "entropy", memBase.Add(-time.Second))
	refresh := w.refresh()
	refresh.Ledger = nil

	execute(t, refresh, "Note.md")
	execute(t, refresh, "Note.md")
	f.fp.ModTime = memBase.Add(-time.Hour)
	f.fp.Size++
	execute(t, refresh, "Note.md")

	if got := len(w.savedPaths()); got != 3 {
		t.Errorf("saved %d times, want 3", got)
	}
	if n := refresh.Ledger.Len(); n != 0 {
		t.Errorf("a missing ledger holds %d notes", n)
	}
}

func TestACancelledRefreshReportsWhatItDidFirst(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	w.addNote("First.md", "entropy", memBase.Add(-time.Hour))
	w.addNote("Second.md", "entropy", memBase.Add(-time.Hour))
	ctx, cancel := context.WithCancel(t.Context())
	w.afterRead = cancel

	res, err := w.refresh().Execute(ctx, memVault, []string{"First.md", "Second.md"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if fmt.Sprint(res.Indexed) != "[First.md]" {
		t.Errorf("indexed %v", res.Indexed)
	}
}

func TestARefreshWithoutALedgerConsumesASettledNoteForCurrent(t *testing.T) {
	t.Parallel()
	w := newMemWorld()
	w.addNote("Note.md", "entropy", memBase.Add(-time.Hour))
	w.index("Note.md")
	refresh := w.refresh()
	refresh.Ledger = nil

	res := execute(t, refresh, "Note.md")

	if fmt.Sprint(res.Indexed) != "[Note.md]" || len(w.savedPaths()) != 0 {
		t.Errorf("indexed %v after %d saves", res.Indexed, len(w.savedPaths()))
	}
}
