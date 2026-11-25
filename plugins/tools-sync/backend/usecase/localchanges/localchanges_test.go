package localchanges_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/localchanges"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

const vault domain.VaultID = "Test"

func newComputer(t *testing.T) *synctest.Computer {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	return synctest.NewFirstComputer(t, server, clock, string(vault), synctest.NewWords(t))
}

func scan(t *testing.T, c *synctest.Computer, bases map[string]string) []transfer.Change {
	t.Helper()
	changes, err := localchanges.NewScanLocalChanges(c.LocalChangesDeps()).Execute(t.Context(), transfer.ScanInput{VaultID: vault, Bases: bases})
	if err != nil {
		t.Fatal(err)
	}
	return changes
}

func getPaths(changes []transfer.Change) []string {
	paths := make([]string, len(changes))
	for i, change := range changes {
		paths[i] = change.Path
	}
	return paths
}

func TestScanLocalChanges_ListsANoteWithItsHashSizeAndTime(t *testing.T) {
	c := newComputer(t)
	c.WriteNote(vault, "Shopping list.md", "milk")

	changes := scan(t, c, nil)

	stat, err := c.Vaults.With(vault).StatFile(t.Context(), "Shopping list.md")
	if err != nil {
		t.Fatal(err)
	}
	want := transfer.Change{Path: "Shopping list.md", Hash: note.Hash([]byte("milk")), Size: 4, ModifiedAt: stat.ModTime.UnixNano()}
	if len(changes) != 1 || changes[0] != want {
		t.Errorf("changes = %+v, want [%+v]", changes, want)
	}
}

func TestScanLocalChanges_ListsTheNotesThatWereThereBeforeTheFirstScan(t *testing.T) {
	c := newComputer(t)
	c.WriteNote(vault, "Plan.md", "tea")
	c.WriteNote(vault, "Food/Shopping list.md", "eggs")

	changes := scan(t, c, nil)

	if paths := getPaths(changes); !slices.Equal(paths, []string{"Food/Shopping list.md", "Plan.md"}) {
		t.Errorf("paths = %v, want both notes, in the order of their paths", paths)
	}
}

func TestScanLocalChanges_ListsANoteChangedAfterAScan(t *testing.T) {
	c := newComputer(t)
	if first := scan(t, c, nil); len(first) != 0 {
		t.Fatalf("changes of an empty vault = %+v, want none", first)
	}
	c.WriteNote(vault, "Shopping list.md", "milk")

	changes := scan(t, c, nil)

	if paths := getPaths(changes); !slices.Equal(paths, []string{"Shopping list.md"}) {
		t.Errorf("paths = %v, want the note written after the first scan", paths)
	}
}

func TestScanLocalChanges_ListsTheLatestContentOfANoteChangedTwice(t *testing.T) {
	c := newComputer(t)
	scan(t, c, nil)
	c.WriteNote(vault, "Shopping list.md", "milk")
	c.WriteNote(vault, "Shopping list.md", "bread")

	changes := scan(t, c, nil)

	if len(changes) != 1 || changes[0].Hash != note.Hash([]byte("bread")) {
		t.Errorf("changes = %+v, want one change with the hash of bread", changes)
	}
}

func TestScanLocalChanges_ListsAFileOfAnyKind(t *testing.T) {
	c := newComputer(t)
	scan(t, c, nil)
	c.WriteNote(vault, "Papers/Report.pdf", "pdf bytes")
	c.WriteNote(vault, "Pictures/Tree.png", "pixels")
	c.WriteNote(vault, "data.bin", "bytes")

	changes := scan(t, c, nil)

	if paths := getPaths(changes); !slices.Equal(paths, []string{"Papers/Report.pdf", "Pictures/Tree.png", "data.bin"}) {
		t.Errorf("paths = %v, want every file that was written, in the order of their paths", paths)
	}
}

func TestScanLocalChanges_IgnoresFilesThatAreNotThePersons(t *testing.T) {
	c := newComputer(t)
	scan(t, c, nil)
	for _, path := range []string{"Papers/Report.pdf.part", "Papers/Report.tmp", "Papers/~$Report.docx", "Notes/Milk.md~", "Pictures/Thumbs.db", "Pictures/desktop.ini"} {
		c.WriteNote(vault, path, "x")
	}
	c.WriteNote(vault, "Shopping list.md", "milk")

	changes := scan(t, c, nil)

	if paths := getPaths(changes); !slices.Equal(paths, []string{"Shopping list.md"}) {
		t.Errorf("paths = %v, want only the note: temporary files and system files are left alone", paths)
	}
}

func TestScanLocalChanges_ListsAFileMovedToAnotherPathAtItsNewPath(t *testing.T) {
	c := newComputer(t)
	c.WriteNote(vault, "Papers/Report.pdf", "pdf bytes")
	hash := note.Hash([]byte("pdf bytes"))
	scan(t, c, map[string]string{"Papers/Report.pdf": hash})
	if _, err := c.Vaults.With(vault).MoveFile(t.Context(), "Papers/Report.pdf", "Papers/Old report.pdf", nil); err != nil {
		t.Fatal(err)
	}

	changes := scan(t, c, map[string]string{"Papers/Report.pdf": hash})

	if paths := getPaths(changes); !slices.Equal(paths, []string{"Papers/Old report.pdf"}) {
		t.Errorf("paths = %v, want the file at its new path, and nothing at the old path", paths)
	}
}

func TestScanLocalChanges_IgnoresARemovedNote(t *testing.T) {
	c := newComputer(t)
	c.WriteNote(vault, "Shopping list.md", "milk")
	scan(t, c, map[string]string{"Shopping list.md": note.Hash([]byte("milk"))})
	c.Vaults.With(vault).Remove("Shopping list.md")

	changes := scan(t, c, map[string]string{"Shopping list.md": note.Hash([]byte("milk"))})

	if len(changes) != 0 {
		t.Errorf("changes = %+v, want none for a removed note", changes)
	}
}

func TestScanLocalChanges_SkipsANoteWhoseHashIsItsBase(t *testing.T) {
	c := newComputer(t)
	scan(t, c, nil)
	c.WriteNote(vault, "Shopping list.md", "milk")

	changes := scan(t, c, map[string]string{"Shopping list.md": note.Hash([]byte("milk"))})

	if len(changes) != 0 {
		t.Errorf("changes = %+v, want none for a note equal to its base", changes)
	}
}

func TestScanLocalChanges_ListsAChangeAgainUntilItsBaseIsSet(t *testing.T) {
	c := newComputer(t)
	scan(t, c, nil)
	c.WriteNote(vault, "Shopping list.md", "milk")
	if first := scan(t, c, nil); len(first) != 1 {
		t.Fatalf("first scan = %+v, want the note", first)
	}

	again := scan(t, c, nil)
	settled := scan(t, c, map[string]string{"Shopping list.md": note.Hash([]byte("milk"))})

	if len(again) != 1 {
		t.Errorf("scan before the base is set = %+v, want the same change again", again)
	}
	if len(settled) != 0 {
		t.Errorf("scan after the base is set = %+v, want none", settled)
	}
}

func TestScanLocalChanges_WaitsForANoteYoungerThanTenSeconds(t *testing.T) {
	c := newComputer(t)
	scan(t, c, nil)
	c.WriteNote(vault, "Shopping list.md", "milk")
	stat, err := c.Vaults.With(vault).StatFile(t.Context(), "Shopping list.md")
	if err != nil {
		t.Fatal(err)
	}
	c.Clock.Set(stat.ModTime.Add(3 * time.Second))

	young := scan(t, c, nil)
	c.Clock.Advance(10 * time.Second)
	later := scan(t, c, nil)

	if len(young) != 0 {
		t.Errorf("changes of a note 3 seconds old = %+v, want none yet", young)
	}
	if paths := getPaths(later); !slices.Equal(paths, []string{"Shopping list.md"}) {
		t.Errorf("changes 13 seconds after the write = %v, want the note without a new change", paths)
	}
}

func TestScanLocalChanges_ListsTheNotesAgainWhenTheFeedStartsAnew(t *testing.T) {
	c := newComputer(t)
	c.WriteNote(vault, "Shopping list.md", "milk")
	hash := note.Hash([]byte("milk"))
	scan(t, c, map[string]string{"Shopping list.md": hash})
	c.Vaults.With(vault).ResetFeed()

	skipped := scan(t, c, map[string]string{"Shopping list.md": hash})
	listed := scan(t, c, nil)

	if len(skipped) != 0 {
		t.Errorf("changes after the reset with the base set = %+v, want none", skipped)
	}
	if paths := getPaths(listed); !slices.Equal(paths, []string{"Shopping list.md"}) {
		t.Errorf("changes after the reset with no base = %v, want the note read from the vault", paths)
	}
}

func TestScanLocalChanges_KeepsItsPositionInFeedJSON(t *testing.T) {
	c := newComputer(t)
	c.WriteNote(vault, "Shopping list.md", "milk")

	scan(t, c, nil)

	var kept map[string]any
	if err := json.Unmarshal(c.ReadStoreFile(t, vault, synctest.FeedFile), &kept); err != nil {
		t.Fatalf("feed.json is not JSON: %v", err)
	}
	if len(kept) == 0 {
		t.Error("feed.json is empty after a scan")
	}
}

type brokenStatHost struct {
	ports.Host
	path   string
	err    error
	onStat func()
}

func (h brokenStatHost) GetFiles(id domain.VaultID) ports.Files {
	return brokenStatFiles{Files: h.Host.GetFiles(id), path: h.path, err: h.err, onStat: h.onStat}
}

type brokenStatFiles struct {
	ports.Files
	path   string
	err    error
	onStat func()
}

func (f brokenStatFiles) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	if path == f.path {
		if f.onStat != nil {
			f.onStat()
		}
		if f.err != nil {
			return domain.FileStat{}, f.err
		}
		return domain.FileStat{}, errors.New("permission denied")
	}
	return f.Files.StatFile(ctx, path)
}

type failingSightings struct{ ports.Sightings }

func (failingSightings) WriteSightings(context.Context, domain.VaultID, map[string]note.Seen) error {
	return errors.New("disk full")
}

func TestScanLocalChanges_ListsTheOtherNotesWhenOneNoteCannotBeRead(t *testing.T) {
	c := newComputer(t)
	c.WriteNote(vault, "Locked.md", "secret")
	c.WriteNote(vault, "Shopping list.md", "milk")
	deps := c.LocalChangesDeps()
	deps.Host = brokenStatHost{Host: c.Host, path: "Locked.md"}

	changes, err := localchanges.NewScanLocalChanges(deps).Execute(t.Context(), transfer.ScanInput{VaultID: vault})
	if err != nil {
		t.Fatal(err)
	}
	later := scan(t, c, nil)

	if paths := getPaths(changes); !slices.Equal(paths, []string{"Shopping list.md"}) {
		t.Errorf("paths = %v, want the readable note", paths)
	}
	if paths := getPaths(later); !slices.Equal(paths, []string{"Locked.md", "Shopping list.md"}) {
		t.Errorf("paths once the note can be read = %v, want both notes", paths)
	}
}

func TestScanLocalChanges_ListsANoteAgainWhenTheSightingsCouldNotBeStored(t *testing.T) {
	c := newComputer(t)
	c.WriteNote(vault, "Shopping list.md", "milk")
	deps := c.LocalChangesDeps()
	deps.State = failingSightings{Sightings: c.State}
	if _, err := localchanges.NewScanLocalChanges(deps).Execute(t.Context(), transfer.ScanInput{VaultID: vault}); err == nil {
		t.Fatal("scan succeeded although the sightings could not be stored")
	}

	changes := scan(t, c, nil)

	if paths := getPaths(changes); !slices.Equal(paths, []string{"Shopping list.md"}) {
		t.Errorf("paths = %v, want the note again", paths)
	}
}

func TestScanLocalChanges_ReturnsTheErrorOfTheContextWhenItEndsWhileTheNotesAreRead(t *testing.T) {
	c := newComputer(t)
	c.WriteNote(vault, "Locked.md", "secret")
	c.WriteNote(vault, "Shopping list.md", "milk")
	ctx, cancel := context.WithCancel(t.Context())
	deps := c.LocalChangesDeps()
	deps.Host = brokenStatHost{Host: c.Host, path: "Locked.md", err: context.Canceled, onStat: cancel}

	_, err := localchanges.NewScanLocalChanges(deps).Execute(ctx, transfer.ScanInput{VaultID: vault})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}
