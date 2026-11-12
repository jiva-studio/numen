package edit_test

import (
	"context"
	"crypto/rand"
	"errors"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
	"github.com/jiva-studio/numen/libs/note/markdown"
	"github.com/jiva-studio/numen/libs/note/ulid"
)

// files are the files of a vault held in memory. A write moves the modification time on, and one held
type files struct {
	content   map[string]string
	kinds     map[string]domain.SourceKind
	clock     time.Time
	stats     map[string]domain.FileStat
	locks     int
	statErr   error
	readErr   error
	lockErr   error
	statCalls int
}

func newFiles(content map[string]string) *files {
	f := &files{
		content: map[string]string{}, kinds: map[string]domain.SourceKind{}, stats: map[string]domain.FileStat{},
		clock: time.Unix(1_700_000_000, 0),
	}
	for path, body := range content {
		f.put(path, body)
	}
	return f
}

func (f *files) put(path, body string) domain.FileStat {
	f.clock = f.clock.Add(time.Second)
	f.content[path] = body
	f.stats[path] = domain.FileStat{Path: path, Kind: domain.KindNote, Size: int64(len(body)), ModTime: f.clock}
	return f.stats[path]
}

func (f *files) StatFile(_ context.Context, path string) (domain.FileStat, error) {
	f.statCalls++
	if f.statErr != nil {
		return domain.FileStat{}, f.statErr
	}
	fp, there := f.stats[path]
	if !there {
		return domain.FileStat{}, fs.ErrNotExist
	}
	if kind, set := f.kinds[path]; set {
		fp.Kind = kind
	}
	return fp, nil
}

func (f *files) Read(_ context.Context, path string) ([]byte, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	body, there := f.content[path]
	if !there {
		return nil, fs.ErrNotExist
	}
	return []byte(body), nil
}

func (f *files) Write(_ context.Context, path string, content []byte, stat domain.FileStat) (domain.FileStat, error) {
	if !stat.IsZero() && !f.stats[path].IsUnchanged(stat) {
		return domain.FileStat{}, edit.ErrStale
	}
	return f.put(path, string(content)), nil
}

func (f *files) Lock(context.Context) (func(), error) {
	if f.lockErr != nil {
		return nil, f.lockErr
	}
	f.locks++
	return func() { f.locks-- }, nil
}

func getVault() domain.Vault {
	return domain.Vault{ID: "v1", Name: "Physics", Path: "/vaults/Physics"}
}

type fixedClock time.Time

func (c fixedClock) Now() time.Time { return time.Time(c) }

func editing(f *files) edit.Edit {
	return edit.Edit{
		Open:    func(domain.Vault) (edit.Files, error) { return f, nil },
		Clock:   fixedClock(time.Unix(1_700_000_000, 0)),
		Entropy: rand.Reader,
	}
}

func TestANoteIsReadAsItsProse(t *testing.T) {
	f := newFiles(map[string]string{"A.md": "---\ntitle: A\n---\n\nhello\r\nworld\n"})

	got, err := edit.Read(t.Context(), f, "A.md")

	if err != nil || got.Outcome != edit.Ok || got.Body != "\nhello\nworld\n" || got.FileStat.Path != "A.md" {
		t.Errorf("got %+v (%v)", got, err)
	}
}

func TestAReadReturnsWhatIsWrongWithTheNoteAndNotAnError(t *testing.T) {
	big := newFiles(map[string]string{"Big.md": strings.Repeat("x", edit.MaxBytes+1)})
	binary := newFiles(map[string]string{"Bin.md": "\xff\xfe"})
	broken := newFiles(map[string]string{"Broken.md": "---\n: [\n---\nbody"})
	book := newFiles(map[string]string{"B.epub": "x"})
	book.kinds["B.epub"] = domain.KindFile
	cases := []struct {
		name  string
		files *files
		path  string
		want  edit.ReadOutcome
	}{
		{"nothing there", newFiles(nil), "None.md", edit.Missing},
		{"too large", big, "Big.md", edit.TooLarge},
		{"not text", binary, "Bin.md", edit.NotText},
		{"frontmatter that is not YAML", broken, "Broken.md", edit.Unreadable},
		{"not a note", book, "B.epub", edit.NotANote},
	}
	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			got, err := edit.Read(t.Context(), one.files, one.path)

			if err != nil || got.Outcome != one.want || got.Body != "" {
				t.Errorf("got %+v (%v)", got, err)
			}
		})
	}
}

func TestApply_StampsIDAndLocksWrite(t *testing.T) {
	f := newFiles(map[string]string{"A.md": "# A\n"})
	read, err := f.StatFile(t.Context(), "A.md")
	if err != nil {
		t.Fatal(err)
	}

	res, err := editing(f).Apply(t.Context(), getVault(), "A.md", func(doc *markdown.Document) error { return doc.SetBody("# B\n") })

	if err != nil || res.IsUnchanged(read) || res.Path != "A.md" || res.Size == 0 {
		t.Fatalf("got %+v (%v)", res, err)
	}
	if got := f.content["A.md"]; !strings.Contains(got, "id:") || !strings.HasSuffix(got, "# B\n") {
		t.Errorf("the note is now %q", got)
	}
	if f.locks != 0 {
		t.Errorf("%d locks were left held", f.locks)
	}
}

func TestAnEditOverANoteThatMovedIsRefused(t *testing.T) {
	f := newFiles(map[string]string{"A.md": "# A\n"})
	read, err := f.StatFile(t.Context(), "A.md")
	if err != nil {
		t.Fatal(err)
	}
	f.put("A.md", "# A, edited by somebody\n")
	e := editing(f)
	e.FileStat = read

	_, err = e.Apply(t.Context(), getVault(), "A.md", func(doc *markdown.Document) error { return doc.SetBody("# B\n") })

	if !errors.Is(err, edit.ErrStale) || f.content["A.md"] != "# A, edited by somebody\n" {
		t.Errorf("got %v and the note %q", err, f.content["A.md"])
	}
}

func TestABaselineOfTheSameProseLandsWhateverTheFileReturns(t *testing.T) {
	f := newFiles(map[string]string{"A.md": "# A\n"})
	read, err := f.StatFile(t.Context(), "A.md")
	if err != nil {
		t.Fatal(err)
	}
	f.put("A.md", "# A\n")
	e := editing(f)
	e.ShouldOverwrite = true
	e.Baseline = &edit.Baseline{Prose: "# A\n", FileStat: read}

	_, err = e.Apply(t.Context(), getVault(), "A.md", func(doc *markdown.Document) error { return doc.SetBody("# B\n") })

	if err != nil || f.content["A.md"] != "# B\n" {
		t.Errorf("got %v and the note %q", err, f.content["A.md"])
	}
}

func TestABaselineOfOtherProseIsStale(t *testing.T) {
	f := newFiles(map[string]string{"A.md": "# A, changed\n"})
	e := editing(f)
	e.ShouldOverwrite = true
	e.Baseline = &edit.Baseline{Prose: "# A\n", FileStat: domain.FileStat{Path: "A.md", Size: 3}}

	_, err := e.Apply(t.Context(), getVault(), "A.md", func(doc *markdown.Document) error { return doc.SetBody("# B\n") })

	if !errors.Is(err, edit.ErrStale) {
		t.Errorf("got %v", err)
	}
}

func TestApply_OverwriteCreatesMissingNoteElseErrors(t *testing.T) {
	f := newFiles(nil)
	e := editing(f)
	e.ShouldOverwrite = true
	if _, err := e.Apply(t.Context(), getVault(), "New.md", func(doc *markdown.Document) error { return doc.SetBody("# New\n") }); err != nil {
		t.Fatal(err)
	}

	_, err := editing(f).Apply(t.Context(), getVault(), "Other.md", func(doc *markdown.Document) error { return doc.SetBody("x") })

	if f.content["New.md"] != "# New\n" || !errors.Is(err, edit.ErrNoNote) {
		t.Errorf("the new note is %q, and editing a missing one gave %v", f.content["New.md"], err)
	}
}

func TestEdit_EnforcesSizeLimit(t *testing.T) {
	f := newFiles(map[string]string{"A.md": "# A\n"})
	e := editing(f)
	e.Bound = 10

	_, err := e.Apply(t.Context(), getVault(), "A.md", func(doc *markdown.Document) error { return doc.SetBody(strings.Repeat("x", 50)) })

	if !errors.Is(err, edit.ErrTooLarge) || f.content["A.md"] != "# A\n" {
		t.Errorf("got %v and the note %q", err, f.content["A.md"])
	}
}

func TestAWriteThatLandedIsReportedAlongsideAnIndexUpdateThatFailed(t *testing.T) {
	f := newFiles(map[string]string{"A.md": "# A\n"})
	e := editing(f)
	busyErr := errors.New("index is busy")
	e.Index = func(context.Context, domain.Vault, []string) error { return busyErr }

	res, err := e.Apply(t.Context(), getVault(), "A.md", func(doc *markdown.Document) error { return doc.SetBody("# B\n") })

	if !errors.Is(err, edit.ErrIndexOutdated) || !errors.Is(err, busyErr) || res.Size == 0 || !strings.HasSuffix(f.content["A.md"], "# B\n") {
		t.Errorf("got %+v (%v)", res, err)
	}
}

func TestANoteIsPointedAtAnotherBelowAType(t *testing.T) {
	f := newFiles(map[string]string{"Sample.md": "---\ntype: sample\n---\n\n# Sample\n"})
	e := editing(f)

	_, err := e.PointAt(t.Context(), getVault(), "Sample.md", "preset", domain.Address{Scheme: domain.SchemeName, Value: "Slow"}, domain.RoleRef)

	if err != nil || !strings.Contains(f.content["Sample.md"], "Slow") {
		t.Errorf("got %v and the note %q", err, f.content["Sample.md"])
	}
}

type names map[string][]string

func (n names) GetNamedPaths(_ context.Context, _ domain.VaultID, name string) ([]string, error) {
	return n[name], nil
}

func TestANoteIsAddressedByItsNameUnlessAnotherSharesIt(t *testing.T) {
	shared := names{"Slow": {"a/Slow.md", "b/Slow.md"}, "Fast": {"Fast.md"}}

	byName, err := edit.GetAddress(t.Context(), shared, "v1", "Fast.md")
	if err != nil {
		t.Fatal(err)
	}
	byPath, err := edit.GetAddress(t.Context(), shared, "v1", "a/Slow.md")
	if err != nil {
		t.Fatal(err)
	}
	_, unreachable := edit.GetAddress(t.Context(), shared, "v1", "a/C#.md")

	if byName.Value != "Fast" || byPath.Value != "a/Slow" || !errors.Is(unreachable, edit.ErrUnaddressable) {
		t.Errorf("got %v and %v, and %v", byName, byPath, unreachable)
	}
}

// Write indexes the note even when the caller cancels its context after the write.
func TestWriteIndexesNoteAfterCallerCancels(t *testing.T) {
	f := newFiles(map[string]string{"A.md": "# A\n"})
	ctx, cancel := context.WithCancel(t.Context())
	var indexErr error
	e := editing(f)
	e.Index = func(ctx context.Context, _ domain.Vault, _ []string) error {
		indexErr = ctx.Err()
		return indexErr
	}

	_, err := e.Apply(ctx, getVault(), "A.md", func(doc *markdown.Document) error {
		cancel()
		return doc.SetBody("# B\n")
	})

	if err != nil || indexErr != nil {
		t.Fatalf("the index was given a context that ended (%v), and the edit answered %v", indexErr, err)
	}
	if got := f.content["A.md"]; !strings.HasSuffix(got, "# B\n") {
		t.Errorf("the note is now %q", got)
	}
}

type failingNames struct{ err error }

func (f failingNames) GetNamedPaths(context.Context, domain.VaultID, string) ([]string, error) {
	return nil, f.err
}

func TestGetAddressBoundariesAndErrors(t *testing.T) {
	shared := names{
		"Fast":    {"sub/Fast.md"},
		".hidden": {"a/.hidden", "b/.hidden"},
		"x":       {"a/x.md", "b/x.md"},
	}
	if _, err := edit.GetAddress(t.Context(), shared, "v1", ""); err == nil {
		t.Error("expected error for empty path")
	}
	wantErr := errors.New("db error")
	if _, err := edit.GetAddress(t.Context(), failingNames{wantErr}, "v1", "Fast.md"); !errors.Is(err, wantErr) {
		t.Errorf("got %v, want %v", err, wantErr)
	}
	gotSingleInSub, err := edit.GetAddress(t.Context(), shared, "v1", "sub/Fast.md")
	if err != nil || gotSingleInSub.Value != "Fast" {
		t.Errorf("got %+v, %v; want Fast", gotSingleInSub, err)
	}
	gotHidden, err := edit.GetAddress(t.Context(), shared, "v1", "a/.hidden")
	if err != nil || gotHidden.Value != "a/.hidden" {
		t.Errorf("got %+v, %v; want a/.hidden", gotHidden, err)
	}
	gotOneChar, err := edit.GetAddress(t.Context(), shared, "v1", "a/x.md")
	if err != nil || gotOneChar.Value != "a/x" {
		t.Errorf("got %+v, %v; want a/x", gotOneChar, err)
	}
}

func TestPointAtValidationAndRemoval(t *testing.T) {
	f := newFiles(map[string]string{"Sample.md": "---\ntype: sample\nlinks:\n  - to: Slow\n    role: ref\n    type: preset\n---\n\n# Sample\n"})
	e := editing(f)

	if _, err := e.PointAt(t.Context(), getVault(), "Sample.md", "", domain.Address{Scheme: domain.SchemeName, Value: "Slow"}, domain.RoleRef); err == nil {
		t.Error("expected error for empty link type")
	}
	if _, err := e.PointAt(t.Context(), getVault(), "Sample.md", "preset", domain.Address{Scheme: domain.SchemeName, Value: "Slow"}, "invalid-role"); err == nil {
		t.Error("expected error for invalid role")
	}
	if _, err := e.PointAt(t.Context(), getVault(), "Sample.md", "preset", domain.Address{}, ""); err != nil {
		t.Errorf("removing link failed: %v", err)
	}
	if strings.Contains(f.content["Sample.md"], "Slow") {
		t.Errorf("link was not removed: %q", f.content["Sample.md"])
	}
}

func TestReadBoundariesAndErrors(t *testing.T) {
	exactMax := newFiles(map[string]string{"Exact.md": strings.Repeat("a", edit.MaxBytes)})
	got, err := edit.Read(t.Context(), exactMax, "Exact.md")
	if err != nil || got.Outcome != edit.Ok || len(got.Body) != edit.MaxBytes {
		t.Errorf("exact MaxBytes: got outcome %q, len %d, err %v", got.Outcome, len(got.Body), err)
	}

	statNotANote := newFiles(nil)
	statNotANote.statErr = edit.ErrNotANote
	got, err = edit.Read(t.Context(), statNotANote, "X.bin")
	if err != nil || got.Outcome != edit.NotANote {
		t.Errorf("ErrNotANote from StatFile: got %+v, %v", got, err)
	}

	statDiskErr := newFiles(nil)
	statDiskErr.statErr = errors.New("stat disk failure")
	if _, err := edit.Read(t.Context(), statDiskErr, "A.md"); !errors.Is(err, statDiskErr.statErr) {
		t.Errorf("StatFile error: got %v", err)
	}

	readDiskErr := newFiles(map[string]string{"A.md": "hello"})
	readDiskErr.readErr = errors.New("read disk failure")
	if _, err := edit.Read(t.Context(), readDiskErr, "A.md"); !errors.Is(err, readDiskErr.readErr) {
		t.Errorf("Read error: got %v", err)
	}

	// File missing during StatFile (`target == false`) but present during Read: treated as NotANote.
	unindexed := newFiles(nil)
	unindexed.content["Ghost.md"] = "hello"
	got, err = edit.Read(t.Context(), unindexed, "Ghost.md")
	if err != nil || got.Outcome != edit.NotANote {
		t.Errorf("unindexed file: got %+v, %v", got, err)
	}
}

func TestCheckSizeAndVocabulary(t *testing.T) {
	if err := edit.CheckSize("A.md", 10, 0); err != nil {
		t.Errorf("maxSize 0: %v", err)
	}
	if err := edit.CheckSize("A.md", 5, 10); err != nil {
		t.Errorf("size < maxSize: %v", err)
	}
	if err := edit.CheckSize("A.md", 10, 10); err != nil {
		t.Errorf("size == maxSize: %v", err)
	}
	if err := edit.CheckSize("A.md", 2, 1); !errors.Is(err, edit.ErrTooLarge) {
		t.Errorf("size 2 > maxSize 1: got %v", err)
	}
	if !edit.IsNoNote(fs.ErrNotExist) || !edit.IsNoNote(edit.ErrNotANote) || edit.IsNoNote(errors.New("other")) {
		t.Error("IsNoNote mismatch")
	}
}

func TestEditSpliceAndHelperBranches(t *testing.T) {
	openErr := errors.New("open failed")
	eOpen := edit.Edit{Open: func(domain.Vault) (edit.Files, error) { return nil, openErr }}
	if _, err := eOpen.Apply(t.Context(), getVault(), "A.md", func(*markdown.Document) error { return nil }); !errors.Is(err, openErr) {
		t.Errorf("Open error: got %v", err)
	}

	fLock := newFiles(map[string]string{"A.md": "# A\n"})
	fLock.lockErr = errors.New("lock failed")
	if _, err := editing(fLock).Apply(t.Context(), getVault(), "A.md", func(*markdown.Document) error { return nil }); !errors.Is(err, fLock.lockErr) {
		t.Errorf("Lock error: got %v", err)
	}

	// Baseline with matching FileStat and different Prose is not stale.
	fSameStat := newFiles(map[string]string{"A.md": "# A\n"})
	statA, err := fSameStat.StatFile(t.Context(), "A.md")
	if err != nil {
		t.Fatalf("StatFile: %v", err)
	}
	eSameStat := editing(fSameStat)
	eSameStat.Baseline = &edit.Baseline{Prose: "different prose", FileStat: statA}
	eSameStat.FileStat = statA
	fSameStat.statCalls = 0
	if _, err := eSameStat.Apply(t.Context(), getVault(), "A.md", func(doc *markdown.Document) error { return doc.SetBody("# B\n") }); err != nil {
		t.Errorf("matching FileStat in Baseline should not be stale: %v", err)
	}
	if fSameStat.statCalls != 1 {
		t.Errorf("expected StatFile to be called when Baseline != nil, got %d", fSameStat.statCalls)
	}

	// When Baseline == nil and FileStat is non-zero, StatFile is not called.
	fNoStat := newFiles(map[string]string{"A.md": "# A\n"})
	statNoStat, err := fNoStat.StatFile(t.Context(), "A.md")
	if err != nil {
		t.Fatalf("StatFile: %v", err)
	}
	fNoStat.statErr = errors.New("StatFile must not be called")
	eNoStat := editing(fNoStat)
	eNoStat.FileStat = statNoStat
	if _, err := eNoStat.Apply(t.Context(), getVault(), "A.md", func(doc *markdown.Document) error { return doc.SetBody("# B\n") }); err != nil {
		t.Errorf("StatFile should be skipped when FileStat is set and Baseline is nil: %v", err)
	}

	// When Baseline == nil and ShouldOverwrite is true, StatFile is not called.
	fOverwriteNoStat := newFiles(map[string]string{"A.md": "# A\n"})
	fOverwriteNoStat.statErr = errors.New("StatFile must not be called")
	eOverwriteNoStat := editing(fOverwriteNoStat)
	eOverwriteNoStat.ShouldOverwrite = true
	if _, err := eOverwriteNoStat.Apply(t.Context(), getVault(), "A.md", func(doc *markdown.Document) error { return doc.SetBody("# B\n") }); err != nil {
		t.Errorf("StatFile should be skipped when ShouldOverwrite is true and Baseline is nil: %v", err)
	}

	// Missing note with ShouldOverwrite: true and non-empty Baseline creates the note.
	fMissingBaseline := newFiles(nil)
	eMissingBaseline := editing(fMissingBaseline)
	eMissingBaseline.ShouldOverwrite = true
	eMissingBaseline.Baseline = &edit.Baseline{Prose: "# Old\n", FileStat: domain.FileStat{Path: "New.md", Size: 6}}
	if _, err := eMissingBaseline.Apply(t.Context(), getVault(), "New.md", func(doc *markdown.Document) error { return doc.SetBody("# Created\n") }); err != nil {
		t.Errorf("creating missing note with Baseline failed: %v", err)
	}

	// Missing note with ShouldOverwrite: false and non-zero FileStat fails in readNote with "read ...: ErrNoNote".
	fReadMissing := newFiles(nil)
	eReadMissing := editing(fReadMissing)
	eReadMissing.FileStat = domain.FileStat{Path: "Missing.md", Size: 1}
	_, err = eReadMissing.Apply(t.Context(), getVault(), "Missing.md", func(*markdown.Document) error { return nil })
	if !errors.Is(err, edit.ErrNoNote) || !strings.HasPrefix(err.Error(), "read Missing.md:") {
		t.Errorf("expected read Missing.md ErrNoNote, got %v", err)
	}

	// ShouldOverwrite: true with non-ErrNotExist read error fails in readNote.
	fReadIOErr := newFiles(nil)
	fReadIOErr.readErr = errors.New("disk read error")
	eReadIOErr := editing(fReadIOErr)
	eReadIOErr.ShouldOverwrite = true
	if _, err := eReadIOErr.Apply(t.Context(), getVault(), "A.md", func(*markdown.Document) error { return nil }); !errors.Is(err, fReadIOErr.readErr) {
		t.Errorf("expected readErr, got %v", err)
	}

	// StatFile fails while Read succeeds (e.g. FileStat zero, ShouldOverwrite false): getExpectedStat returns "look at ...".
	fStatFail := newFiles(map[string]string{"A.md": "# A\n"})
	fStatFail.statErr = fs.ErrNotExist
	_, err = editing(fStatFail).Apply(t.Context(), getVault(), "A.md", func(*markdown.Document) error { return nil })
	if !errors.Is(err, edit.ErrNoNote) || !strings.HasPrefix(err.Error(), "look at A.md:") {
		t.Errorf("expected look at A.md ErrNoNote, got %v", err)
	}

	// KindFile fails in getExpectedStat with ErrNotANote.
	fKindFile := newFiles(map[string]string{"Book.epub": "content"})
	fKindFile.kinds["Book.epub"] = domain.KindFile
	_, err = editing(fKindFile).Apply(t.Context(), getVault(), "Book.epub", func(*markdown.Document) error { return nil })
	if !errors.Is(err, edit.ErrNotANote) {
		t.Errorf("expected ErrNotANote, got %v", err)
	}

	// Empty Kind in FileStat succeeds in getExpectedStat, and `on` is passed to Write for staleness check.
	fEmptyKind := newFiles(map[string]string{"A.md": "# A\n"})
	fEmptyKind.kinds["A.md"] = ""
	_, err = editing(fEmptyKind).Apply(t.Context(), getVault(), "A.md", func(doc *markdown.Document) error {
		// Simulate concurrent modification during change so Write checks `against` (`on`).
		fEmptyKind.put("A.md", "# Concurrent\n")
		return doc.SetBody("# B\n")
	})
	if !errors.Is(err, edit.ErrStale) {
		t.Errorf("expected ErrStale when file changed between statNote and Write, got %v", err)
	}

	// Unreadable frontmatter fails in markdown.Open during Apply.
	fBadYAML := newFiles(map[string]string{"Bad.md": "---\n: [\n---\nbody"})
	if _, err := editing(fBadYAML).Apply(t.Context(), getVault(), "Bad.md", func(*markdown.Document) error { return nil }); !errors.Is(err, edit.ErrUnreadable) {
		t.Errorf("expected ErrUnreadable, got %v", err)
	}

	// Error from change callback is returned wrapped with path.
	fChangeErr := newFiles(map[string]string{"A.md": "# A\n"})
	changeErr := errors.New("change failed")
	_, err = editing(fChangeErr).Apply(t.Context(), getVault(), "A.md", func(*markdown.Document) error { return changeErr })
	if !errors.Is(err, changeErr) || !strings.HasPrefix(err.Error(), "A.md:") {
		t.Errorf("expected wrapped changeErr, got %v", err)
	}

	// Error from addIdentifier when Entropy is nil.
	fNoEntropy := newFiles(map[string]string{"A.md": "# A\n"})
	eNoEntropy := editing(fNoEntropy)
	eNoEntropy.Entropy = nil
	if _, err := eNoEntropy.Apply(t.Context(), getVault(), "A.md", func(*markdown.Document) error { return nil }); !errors.Is(err, ulid.ErrNoEntropy) {
		t.Errorf("expected ErrNoEntropy, got %v", err)
	}
}
