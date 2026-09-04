package filesystem_test

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ulid"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

func walkPaths(t *testing.T, root string) []string {
	t.Helper()
	src, err := filesystem.Open(root, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := src.Walk(t.Context(), func(r domain.Fingerprint) error {
		got = append(got, r.Path)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	return got
}

// TestWalkReportsEverySourceAndNothingElse. A walk reports the files something
// reads and no others: the note in the hidden folder and the service folder are
// the vault's and not the application's to index.
func TestWalkReportsEverySourceAndNothingElse(t *testing.T) {
	got := walkPaths(t, testsupport.VaultDir(t))
	want := []string{
		"Thermodynamics.md",
		"assets/paper.pdf",
		"cards/Animal.md",
		"cards/Animals.md",
		"cards/Term.md",
		"cards/crlf-deck.md",
		"cards/empty-deck.md",
		"cards/fieldless-stencil.md",
		"daily/2026-08-15.md",
		"edge/broken-frontmatter.md",
		"edge/broken-links.md",
		"edge/code-fence.md",
		"edge/crlf.md",
		"edge/unicode.md",
		"notes/Entropy.md",
	}
	if !slices.Equal(got, want) {
		t.Errorf("walk found\n  %v\nwant\n  %v", got, want)
	}
}

// walkedKinds is what a walk found, as the kind of source each path is.
func walkedKinds(t *testing.T, root string, opts filesystem.Options) map[string]domain.SourceKind {
	t.Helper()
	src, err := filesystem.Open(root, opts)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]domain.SourceKind{}
	if err := src.Walk(t.Context(), func(r domain.Fingerprint) error {
		got[r.Path] = r.Kind
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return got
}

// TestWalkSaysWhichKindEachSourceIs. A walk reports the kind of each source, so
// a book is never handed to the markdown parser.
func TestWalkSaysWhichKindEachSourceIs(t *testing.T) {
	root := testsupport.CopyVault(t)
	testsupport.WriteBook(t, root, "library/A Book.epub")

	kinds := walkedKinds(t, root, filesystem.Options{})

	if got := kinds["library/A Book.epub"]; got != domain.KindBook {
		t.Errorf("the book walked as %q", got)
	}
	if got := kinds["notes/Entropy.md"]; got != domain.KindNote {
		t.Errorf("the note walked as %q", got)
	}
	notes := 0
	for _, kind := range kinds {
		if kind == domain.KindNote {
			notes++
		}
	}
	if notes != 14 {
		t.Errorf("walked %d notes, want the 14 the fixture holds", notes)
	}
}

// TestAFormatNothingReadsIsNotASource. Which files are sources is decided by
// type, so a file of a format no reader handles is reported by nothing and
// stats as absent, however large a part of the vault it is.
func TestAFormatNothingReadsIsNotASource(t *testing.T) {
	root := testsupport.CopyVault(t)
	testsupport.WriteBook(t, root, "library/A Book.epub")
	scanned := "assets/scan.png"
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(scanned)), []byte("PNG"), 0o644); err != nil {
		t.Fatal(err)
	}

	if kind, found := walkedKinds(t, root, filesystem.Options{})[scanned]; found {
		t.Errorf("walk reported the image as %q", kind)
	}

	src, err := filesystem.Open(root, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.Stat(t.Context(), scanned); !errors.Is(err, port.ErrNotANote) {
		t.Errorf("stat of the image gave %v, want ErrNotANote", err)
	}
}

// TestStatDoesNotSayABookIsGone. A refresh reads fs.ErrNotExist as "the file was
// removed", so a book the vault holds has to stat like the note beside it.
func TestStatDoesNotSayABookIsGone(t *testing.T) {
	root := testsupport.CopyVault(t)
	testsupport.WriteBook(t, root, "library/A Book.epub")

	src, err := filesystem.Open(root, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := src.Stat(t.Context(), "library/A Book.epub")
	if err != nil {
		t.Fatalf("stat of a book the vault holds: %v", err)
	}
	if ref.Kind != domain.KindBook {
		t.Errorf("stat says the book is %q", ref.Kind)
	}
	if ref.Size <= 0 || ref.ModTime == 0 {
		t.Errorf("stat gave %+v", ref)
	}
	if _, err := src.Read(t.Context(), "library/A Book.epub"); err != nil {
		t.Errorf("read of a book the vault holds: %v", err)
	}
}

// TestABookInAHiddenFolderIsNotASourceEither. One set of rules answers for every
// kind: what the vault says to leave alone is left alone whatever is in it.
func TestABookInAHiddenFolderIsNotASourceEither(t *testing.T) {
	root := testsupport.CopyVault(t)
	testsupport.WriteBook(t, root, ".obsidian/A Book.epub")

	kinds := walkedKinds(t, root, filesystem.Options{})
	if kind, found := kinds[".obsidian/A Book.epub"]; found {
		t.Errorf("walk reported a book from a hidden folder as %q", kind)
	}

	src, err := filesystem.Open(root, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.Stat(t.Context(), ".obsidian/A Book.epub"); !errors.Is(err, port.ErrNotANote) {
		t.Errorf("stat gave %v, want ErrNotANote", err)
	}
}

func TestWalkSkipsTheServiceFolderAndOtherToolsFolders(t *testing.T) {
	for _, p := range walkPaths(t, testsupport.VaultDir(t)) {
		if p[0] == '.' {
			t.Errorf("walk reported %q from a hidden folder", p)
		}
	}
}

func TestWalkPathsAreRelativeAndSlashed(t *testing.T) {
	// The index describes the same note on every platform, so a walk may not
	// leak an absolute path or a backslash into it.
	for _, p := range walkPaths(t, testsupport.VaultDir(t)) {
		if filepath.IsAbs(p) {
			t.Errorf("%q is absolute", p)
		}
		for _, r := range p {
			if r == '\\' {
				t.Errorf("%q contains a backslash", p)
			}
		}
	}
}

func TestWalkReportsSizeAndTime(t *testing.T) {
	src, err := filesystem.Open(testsupport.VaultDir(t), filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := src.Walk(t.Context(), func(r domain.Fingerprint) error {
		if r.Size <= 0 {
			t.Errorf("%s has size %d", r.Path, r.Size)
		}
		if r.ModTime == 0 {
			t.Errorf("%s has no modification time", r.Path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// listing is the names of what one folder holds, in the order they came back.
func listing(t *testing.T, src *filesystem.VaultReader, folder string) []string {
	t.Helper()
	entries, err := src.List(t.Context(), folder)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name)
	}
	return names
}

// A listing holds every file of the folder and every folder under it, and what
// the vault says to leave alone is left out. Folders come first, and names are
// compared without regard to case.
func TestAListingIsOrderedAndLeavesTheHiddenOut(t *testing.T) {
	v := testsupport.NewVault(t, map[string]string{
		"alpha/Entropy.md":       "# Entropy\n",
		"Zulu/Heat.md":           "# Heat\n",
		"Banana.md":              "# Banana\n",
		".secret.md":             "# Secret\n",
		".obsidian/workspace.md": "{}\n",
	})
	laid(t, v.Path, "apple.txt")

	src, err := filesystem.Open(v.Path, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alpha", "Zulu", "apple.txt", "Banana.md"}
	if got := listing(t, src, ""); !slices.Equal(got, want) {
		t.Errorf("the root listed as\n  %v\nwant\n  %v", got, want)
	}
}

// A listing says which kind of source each file is, and names a file nothing
// reads all the same. A folder is of no kind and is what a listing of its own
// answers to.
func TestAListingSaysWhatEachEntryIs(t *testing.T) {
	v := testsupport.NewVault(t, map[string]string{
		"library/Notes.md":      "# Notes\n",
		"library/deeper/Old.md": "# Old\n",
	})
	laid(t, v.Path, "library/A Book.epub")
	laid(t, v.Path, "library/scan.png")

	src, err := filesystem.Open(v.Path, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}
	entries, err := src.List(t.Context(), "library")
	if err != nil {
		t.Fatal(err)
	}

	want := []domain.Entry{
		{Path: "library/deeper", Name: "deeper", IsFolder: true},
		{Path: "library/A Book.epub", Name: "A Book.epub", Kind: domain.KindBook},
		{Path: "library/Notes.md", Name: "Notes.md", Kind: domain.KindNote},
		{Path: "library/scan.png", Name: "scan.png"},
	}
	if !slices.Equal(entries, want) {
		t.Errorf("listed %+v, want %+v", entries, want)
	}
}

// The service folder's name is a setting, and a listing leaves it out whether
// the name begins with a dot or not.
func TestAListingLeavesTheServiceFolderOut(t *testing.T) {
	v := testsupport.NewVault(t, map[string]string{"Entropy.md": "# Entropy\n"})
	laid(t, v.Path, "store/state.json")

	src, err := filesystem.Open(v.Path, filesystem.Options{ServiceDir: "store"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Entropy.md"}
	if got := listing(t, src, ""); !slices.Equal(got, want) {
		t.Errorf("the root listed as\n  %v\nwant\n  %v", got, want)
	}
}

// A path that leaves the vault is refused, and a folder that is not there is
// answered as what it is.
func TestAListingOfWhatTheVaultHasNotIsRefused(t *testing.T) {
	v := testsupport.NewVault(t, map[string]string{
		"Entropy.md":   "# Entropy\n",
		"Notes.txt":    "a list\n",
		"physics/x.md": "# X\n",
	})
	src, err := filesystem.Open(v.Path, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}

	for _, folder := range []string{"../", "notes/../../elsewhere"} {
		if _, err := src.List(t.Context(), folder); !errors.Is(err, filesystem.ErrOutside) {
			t.Errorf("list %q: want ErrOutside, got %v", folder, err)
		}
	}
	if _, err := src.List(t.Context(), "nowhere"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("list of a folder that is not there: want fs.ErrNotExist, got %v", err)
	}
	// A listing is of a folder. A path holding a file of any kind holds none.
	for _, folder := range []string{"Entropy.md", "Notes.txt", "physics/x.md"} {
		if _, err := src.List(t.Context(), folder); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("list of the file at %q: want fs.ErrNotExist, got %v", folder, err)
		}
	}
	if _, err := src.List(t.Context(), filesystem.DefaultServiceDir); err == nil {
		t.Error("the service folder was listed")
	}
}

func TestOpenRejectsWhatIsNotAFolder(t *testing.T) {
	if _, err := filesystem.Open(filepath.Join(testsupport.VaultDir(t), "Thermodynamics.md"), filesystem.Options{}); err == nil {
		t.Error("a file was accepted as a vault")
	}
	if _, err := filesystem.Open(filepath.Join(t.TempDir(), "nope"), filesystem.Options{}); err == nil {
		t.Error("a missing folder was accepted as a vault")
	}
}

func TestReadReturnsTheFile(t *testing.T) {
	src, err := filesystem.Open(testsupport.VaultDir(t), filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := src.Read(t.Context(), "notes/Entropy.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Error("read returned nothing")
	}
}

// TestANoteOverTheBoundIsNotRead. The bound is on the scan as much as on a
// write: a file over it is a file the vault does not hold as a note, and its
// bytes are never in memory. A book is left to the caller reading it.
func TestANoteOverTheBoundIsNotRead(t *testing.T) {
	root := t.TempDir()
	big := make([]byte, 64)
	if err := os.WriteFile(filepath.Join(root, "huge.md"), big, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "paper.pdf"), big, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "small.md"), []byte("# note\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	src, err := filesystem.Open(root, filesystem.Options{MaxNoteBytes: 32})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.Read(t.Context(), "huge.md"); !errors.Is(err, port.ErrNotANote) {
		t.Errorf("read huge.md: %v, want ErrNotANote", err)
	}
	if _, err := src.Read(t.Context(), "small.md"); err != nil {
		t.Errorf("read small.md: %v", err)
	}
	if _, err := src.Read(t.Context(), "paper.pdf"); err != nil {
		t.Errorf("read paper.pdf: %v", err)
	}
}

func TestOpenDoesNotWriteIntoTheFolder(t *testing.T) {
	// Scanning a folder and adding a vault are different acts, and only the
	// second may write.
	dir := t.TempDir()
	if _, err := filesystem.Open(dir, filesystem.Options{}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("opening a folder created %v", entries)
	}
}

func TestInitializeGivesAnIdentityOnceAndKeepsIt(t *testing.T) {
	dir := t.TempDir()

	first, err := filesystem.Initialize(dir, "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !ulid.Valid(first.ID) {
		t.Fatalf("identity %q is not a ULID", first.ID)
	}
	if first.Version != 1 {
		t.Errorf("format version = %d", first.Version)
	}

	// Re-adding a vault must not mint a new identity: every row in the index
	// points at the old one.
	second, err := filesystem.Initialize(dir, "", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Errorf("identity changed from %s to %s", first.ID, second.ID)
	}
}

func TestInitializeWritesOnlyIntoTheServiceFolder(t *testing.T) {
	dir := t.TempDir()
	if _, err := filesystem.Initialize(dir, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filesystem.DefaultServiceDir {
		t.Errorf("initialize touched %v, want only %s", entries, filesystem.DefaultServiceDir)
	}
}

func TestServiceFolderNameIsASetting(t *testing.T) {
	dir := t.TempDir()
	if _, err := filesystem.Initialize(dir, "_numen", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "_numen", "config.json")); err != nil {
		t.Errorf("configured service folder not used: %v", err)
	}
}

func TestReadConfigDistinguishesMissingFromCorrupt(t *testing.T) {
	dir := t.TempDir()
	if _, err := filesystem.ReadConfig(dir, ""); !errors.Is(err, filesystem.ErrNotAVault) {
		t.Errorf("missing configuration gave %v, want ErrNotAVault", err)
	}

	svc := filepath.Join(dir, filesystem.DefaultServiceDir)
	if err := os.MkdirAll(svc, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(svc, "config.json"), []byte(`{"v":1,"id":"nonsense"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// A corrupt identity is not "no vault here": treating it as absent would
	// mint a second identity for a vault that already has one.
	err := func() error { _, e := filesystem.ReadConfig(dir, ""); return e }()
	if err == nil || errors.Is(err, filesystem.ErrNotAVault) {
		t.Errorf("corrupt identity gave %v", err)
	}
	if !errors.Is(err, ulid.ErrInvalid) {
		t.Errorf("error does not say what is wrong: %v", err)
	}
}

func TestANoteIsAMarkdownFile(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.md", "b.markdown", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("# x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if got := walked(t, dir, filesystem.Options{}); !slices.Equal(got, []string{"a.md"}) {
		t.Errorf("the walk found %v, want [a.md]", got)
	}
}

func TestWhichExtensionsAreBooksIsASetting(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.md", "b.epub", "c.pdf"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// The default is every format a reader takes text out of.
	want := map[string]domain.SourceKind{
		"a.md":   domain.KindNote,
		"b.epub": domain.KindBook,
		"c.pdf":  domain.KindBook,
	}
	if got := walkedKinds(t, dir, filesystem.Options{}); !maps.Equal(got, want) {
		t.Errorf("default found %v, want %v", got, want)
	}

	want = map[string]domain.SourceKind{"a.md": domain.KindNote, "c.pdf": domain.KindBook}
	got := walkedKinds(t, dir, filesystem.Options{BookExtensions: []string{".pdf"}})
	if !maps.Equal(got, want) {
		t.Errorf("configured found %v, want %v", got, want)
	}
}

func TestTheConfiguredServiceFolderIsSkippedEvenWithoutALeadingDot(t *testing.T) {
	// The name is a setting because a leading dot is not free — some sync tools
	// skip hidden directories — so a service folder called _numen must be
	// skipped by its name rather than by its shape.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "_numen"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "_numen", "notes.md"), []byte("# ours\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "real.md"), []byte("# theirs\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := walked(t, dir, filesystem.Options{ServiceDir: "_numen"})
	if !slices.Equal(got, []string{"real.md"}) {
		t.Errorf("walk found %v, want [real.md]", got)
	}
}

func walked(t *testing.T, root string, opts filesystem.Options) []string {
	t.Helper()
	src, err := filesystem.Open(root, opts)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := src.Walk(t.Context(), func(r domain.Fingerprint) error {
		got = append(got, r.Path)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	return got
}

// A path that leaves the vault is refused rather than resolved. `filepath.Join`
// would clean the dot-dots away and read whatever it landed on, and a caller
// from outside the application is exactly who would try.
func TestAPathThatLeavesTheVaultIsRefused(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.md"), []byte("not yours"), 0o600); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(outside, "vault")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	reader, err := filesystem.Open(root, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"../secret.md",
		"notes/../../secret.md",
		"./notes/../../secret.md",
	} {
		if _, err := reader.Read(t.Context(), path); !errors.Is(err, filesystem.ErrOutside) {
			t.Errorf("read %q: want ErrOutside, got %v", path, err)
		}
		if _, err := reader.Stat(t.Context(), path); err == nil {
			t.Errorf("stat %q: want a refusal", path)
		}
	}
}

// Writing is a thing only a note is open to. The vault holds a book and an
// attachment, and neither is a file this application puts bytes into.
func TestOnlyANoteIsWrittenTo(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{".git/config", "photo.png", "library/A Book.epub", "notes/keep.md"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.Dir(path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, path), []byte("theirs"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writer, err := filesystem.OpenForWriting(root, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{".git/config", "photo.png", "library/A Book.epub"} {
		if _, err := writer.Write(t.Context(), path, []byte("mine"), domain.Fingerprint{}); !errors.Is(err, filesystem.ErrNotANote) {
			t.Errorf("write %s: want ErrNotANote, got %v", path, err)
		}
		if kept, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(path))); string(kept) != "theirs" {
			t.Errorf("%s was written to anyway", path)
		}
	}

	if _, err := writer.Write(t.Context(), "notes/keep.md", []byte("mine"), domain.Fingerprint{}); err != nil {
		t.Errorf("a note is still writable: %v", err)
	}
}

// A folder inside the vault may be a link to somewhere else, and a write
// through it lands outside. The text of the path says nothing about that.
func TestAWriteDoesNotFollowALinkOutOfTheVault(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.md"), []byte("not yours"), 0o600); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Skipf("this filesystem does not do symlinks: %v", err)
	}
	writer, err := filesystem.OpenForWriting(root, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := writer.Write(t.Context(), "linked/secret.md", []byte("mine"), domain.Fingerprint{}); !errors.Is(err, filesystem.ErrOutside) {
		t.Errorf("want ErrOutside, got %v", err)
	}
	if kept, _ := os.ReadFile(filepath.Join(outside, "secret.md")); string(kept) != "not yours" {
		t.Error("a file outside the vault was written")
	}
}
