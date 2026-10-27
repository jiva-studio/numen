package hosttest_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
)

func TestAWriteFromAnOldReadIsStale(t *testing.T) {
	ctx := context.Background()
	v := hosttest.New("Physics")
	v.File("a.md", "one")
	_, old, _ := v.Read(ctx, "a.md")
	if _, err := v.Write(ctx, "a.md", []byte("two"), old); err != nil {
		t.Fatal(err)
	}

	_, err := v.Write(ctx, "a.md", []byte("three"), old)

	if !errors.Is(err, host.ErrStale) {
		t.Errorf("got %v", err)
	}
}

func TestACreateOverAFileIsOccupied(t *testing.T) {
	v := hosttest.New("Physics")
	v.File("a.md", "one")

	if err := v.Create(context.Background(), "a.md", nil); !errors.Is(err, host.ErrOccupied) {
		t.Errorf("got %v", err)
	}
	if err := v.Create(context.Background(), "../a.md", nil); !errors.Is(err, host.ErrOutside) {
		t.Errorf("got %v", err)
	}
}

func TestAHiddenVaultAnswersNoVault(t *testing.T) {
	v := hosttest.New("Physics")
	v.Hide()

	if _, err := v.GetVault(context.Background()); !errors.Is(err, host.ErrNoVault) {
		t.Errorf("got %v", err)
	}
}

func TestAnotherVaultIsWorkedByItsIdentity(t *testing.T) {
	ctx := context.Background()
	physics, chemistry := hosttest.New("Physics"), hosttest.New("Chemistry")
	chemistry.File("a.md", "x")
	physics.Beside(chemistry)

	recognized, _ := physics.GetVaults(ctx)
	_, _, readErr := physics.With("Chemistry").Read(ctx, "a.md")
	_, _, unknown := physics.With("Biology").Read(ctx, "a.md")

	if len(recognized) != 2 || readErr != nil || !errors.Is(unknown, host.ErrUnknownVault) {
		t.Errorf("the vaults were %v, the read gave %v, an unknown vault gave %v", recognized, readErr, unknown)
	}
}

func TestAMoveIsReportedToTheWatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	v := hosttest.New("Physics")
	moves, err := v.WatchMoves(ctx)
	if err != nil {
		t.Fatal(err)
	}

	v.Move()

	if _, open := <-moves; !open {
		t.Error("the watch closed")
	}
}

const animal = "---\ntype: stencil\ntitle: The animal\nfields:\n  - Name\n---\n\n## Front\n\n{{Name}}\n"

func TestANoteIsInTheIndexAsSoonAsItsFileIsThere(t *testing.T) {
	ctx := context.Background()
	v := hosttest.New("Physics")
	v.File("Animal.md", animal)
	v.File("assets/picture.png", "x")

	notes, _ := v.GetNotes(ctx, []string{"Animal.md", "assets/picture.png", "None.md"})
	types, _ := v.GetTypes(ctx, []string{"Animal.md"})
	stencils, _ := v.GetPathsOfType(ctx, "stencil")

	if len(notes) != 1 || notes["Animal.md"].Title != "The animal" || types["Animal.md"] != "stencil" || len(stencils) != 1 {
		t.Errorf("got %v, %v and %v", notes, types, stencils)
	}
}

func TestANameReachesTheNearestNoteAndReturnsWhenSeveralAnswer(t *testing.T) {
	ctx := context.Background()
	v := hosttest.New("Physics")
	v.File("a/Entropy.md", "# A\n")
	v.File("b/Entropy.md", "# B\n")
	v.File("c/Note.md", "---\nlinks:\n  - to: Entropy\n    role: ref\n---\n\nbody\n")

	reached, _ := v.ResolveLinks(ctx, "c/Note.md", []string{"Entropy", "Missing"})
	named, _ := v.GetNamedPaths(ctx, "entropy")

	got := reached["Entropy"]
	if got.To != "a/Entropy.md" || !got.IsAmbiguous || len(reached) != 1 || len(named) != 2 {
		t.Errorf("got %+v and %v", reached, named)
	}
}

func TestALinkIsABacklinkOnlyWhereItResolves(t *testing.T) {
	ctx := context.Background()
	v := hosttest.New("Physics")
	v.File("Target.md", "# Target\n")
	v.File("Other/Target.md", "# Another target\n")
	v.File("Deck.md", "---\nlinks:\n  - to: Target\n    role: ref\n---\n\nbody\n")

	backlinks, _ := v.GetBacklinks(ctx, "Target.md")
	away, _ := v.GetBacklinks(ctx, "Other/Target.md")
	forward, _ := v.GetLinks(ctx, "Deck.md")

	if len(backlinks) != 1 || backlinks[0].From != "Deck.md" || len(away) != 0 || len(forward) != 1 || forward[0].To != "Target.md" {
		t.Errorf("got %+v, %+v and %+v", backlinks, away, forward)
	}
}

func TestAVaultOnDiskIsChangedByAProgramAdjacentIt(t *testing.T) {
	ctx := context.Background()
	v := hosttest.OnDisk(t, "Physics", map[string]string{"a/Note.md": "# One\n"})
	_, before, _ := v.Read(ctx, "a/Note.md")
	if err := os.WriteFile(filepath.Join(v.Vault().Path, "a", "Note.md"), []byte("# Two, longer\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, writeErr := v.Write(ctx, "a/Note.md", []byte("# Three\n"), before)
	notes, _ := v.GetNotes(ctx, []string{"a/Note.md"})

	if !errors.Is(writeErr, host.ErrStale) || notes["a/Note.md"].Title != "Note" {
		t.Errorf("the write gave %v and the notes %v", writeErr, notes)
	}
}

func TestACreateOnDiskMakesTheFoldersAndRejectsAFileThatIsThere(t *testing.T) {
	ctx := context.Background()
	v := hosttest.OnDisk(t, "Physics", nil)

	first := v.Create(ctx, "decks/New.md", []byte("# New\n"))
	again := v.Create(ctx, "decks/New.md", []byte("# New\n"))

	body, there := v.Content("decks/New.md")
	if first != nil || !errors.Is(again, host.ErrOccupied) || !there || body != "# New\n" {
		t.Errorf("got %v, %v and %q", first, again, body)
	}
}

func TestAFileTheHostMayNotOpenFailsWithWhatItWasReportedToFailWith(t *testing.T) {
	ctx := context.Background()
	v := hosttest.New("Physics")
	v.File("a.md", "x")
	testErr := errors.New("permission denied")
	v.Refuse("a.md", testErr)

	_, _, readErr := v.Read(ctx, "a.md")
	v.Refuse("a.md", nil)
	_, _, again := v.Read(ctx, "a.md")

	if !errors.Is(readErr, testErr) || again != nil {
		t.Errorf("got %v and then %v", readErr, again)
	}
}

func TestTheIndexKnowsAFileChangedAdjacentItOnlyOnceItIsReported(t *testing.T) {
	ctx := context.Background()
	v := hosttest.OnDisk(t, "Physics", map[string]string{"A.md": "---\ntype: deck\n---\n"})
	if err := os.WriteFile(filepath.Join(v.Vault().Path, "B.md"), []byte("---\ntype: deck\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	before, _ := v.GetPathsOfType(ctx, "deck")
	scanErr := v.Scan(ctx, []string{"B.md"})
	after, _ := v.GetPathsOfType(ctx, "deck")

	if len(before) != 1 || scanErr != nil || len(after) != 2 {
		t.Errorf("the index held %v, then %v (%v)", before, after, scanErr)
	}
}

func TestAFileInAHiddenFolderIsRemainingAloneByTheVault(t *testing.T) {
	ctx := context.Background()
	v := hosttest.OnDisk(t, "Physics", map[string]string{"decks/.Hidden.md": "---\ntype: deck\n---\n"})

	_, statErr := v.GetFingerprint(ctx, "decks/.Hidden.md")
	_, missing := v.GetFingerprint(ctx, "decks/Nothing.md")
	decks, _ := v.GetPathsOfType(ctx, "deck")

	if !errors.Is(statErr, host.ErrNotANote) || !errors.Is(missing, host.ErrNotExist) || len(decks) != 0 {
		t.Errorf("got %v, %v and %v", statErr, missing, decks)
	}
}

func TestVaultsMadeKnownToOneAnotherAreKnownToAll(t *testing.T) {
	ctx := context.Background()
	physics, chemistry, biology := hosttest.New("Physics"), hosttest.New("Chemistry"), hosttest.New("Biology")
	biology.File("a.md", "x")
	physics.Beside(chemistry)
	chemistry.Beside(biology)

	_, _, reached := chemistry.With("Physics").Read(ctx, "nothing.md")
	_, _, viaPhysics := physics.With("Biology").Read(ctx, "a.md")

	if !errors.Is(reached, host.ErrNotExist) || viaPhysics != nil {
		t.Errorf("got %v and %v", reached, viaPhysics)
	}
}

func TestAVaultWhoseFolderIsMissingRejectsItsFilesAndItsStore(t *testing.T) {
	ctx := context.Background()
	v := hosttest.OnDisk(t, "Physics", map[string]string{"a.md": "x"})
	if err := os.RemoveAll(v.Vault().Path); err != nil {
		t.Fatal(err)
	}

	_, _, readErr := v.Read(ctx, "a.md")
	appendErr := v.Store().Append(ctx, "log", []byte("x"))
	_, listErr := v.Store().List(ctx)
	notes, _ := v.GetNotes(ctx, []string{"a.md"})

	if readErr == nil || errors.Is(readErr, host.ErrNotExist) || appendErr == nil || listErr == nil || len(notes) != 1 {
		t.Errorf("got %v, %v, %v and %v", readErr, appendErr, listErr, notes)
	}
}

func TestSetSourceKindAppliesToFilesOfTheExtension(t *testing.T) {
	ctx := context.Background()
	v := hosttest.New("Physics")
	v.File("a.sample", "one")
	v.SetSourceKind(".SAMPLE", domain.KindBook)

	fingerprint, err := v.GetFingerprint(ctx, "a.sample")
	if err != nil || fingerprint.Kind != domain.KindBook {
		t.Errorf("fingerprint = %+v, %v, want the kind set for the extension", fingerprint, err)
	}
	v.File("b.other", "two")
	if fingerprint, _ := v.GetFingerprint(ctx, "b.other"); fingerprint.Kind != domain.KindNote {
		t.Errorf("fingerprint = %+v, want a file of an extension nothing set to be a note", fingerprint)
	}
}
