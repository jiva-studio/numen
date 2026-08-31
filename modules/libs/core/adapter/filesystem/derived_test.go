package filesystem_test

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

func store(t *testing.T) (*filesystem.Derived, string) {
	t.Helper()
	root := t.TempDir()
	if _, err := filesystem.Initialize(root, filesystem.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}
	derived, err := filesystem.OpenDerived(root, filesystem.Options{}, filesystem.OCRDir)
	if err != nil {
		t.Fatal(err)
	}
	return derived, root
}

func TestOpeningTheStoreWritesNothing(t *testing.T) {
	// The application looks at every vault it holds. One that puts a folder in
	// each of them has changed a vault by being asked about it.
	derived, root := store(t)
	_ = derived

	at := filepath.Join(root, filesystem.DefaultServiceDir, filesystem.OCRDir)
	if _, err := os.Stat(at); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("opening the store made %s: %v", filesystem.OCRDir, err)
	}
}

// A vault folder taken away under a store that is writing is not a vault to go
// on writing into: creating the parents of a name would put a stub where the
// vault was, and what is written there is shadowed the moment the real one is
// back.
func TestAStoreRefusesAFolderThatIsNoLongerTheVault(t *testing.T) {
	derived, root := store(t)
	ctx := t.Context()
	if err := derived.Append(ctx, "ocr/run.txt", []byte("one\n")); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}

	if err := derived.Append(ctx, "ocr/run.txt", []byte("two\n")); !errors.Is(err, filesystem.ErrNotThisVault) {
		t.Errorf("appending into a vault that is gone gave %v", err)
	}
	if err := derived.Write(ctx, "ocr/run.txt", []byte("two\n")); !errors.Is(err, filesystem.ErrNotThisVault) {
		t.Errorf("writing into a vault that is gone gave %v", err)
	}
	if _, err := derived.List(ctx, filesystem.OCRDir); !errors.Is(err, filesystem.ErrNotThisVault) {
		t.Errorf("listing a vault that is gone gave %v", err)
	}
	if _, err := os.Stat(root); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the vault folder was made again to write into: %v", err)
	}

	// A folder put back at the path, carrying an identity of its own, is
	// another vault and not this one.
	if _, err := filesystem.Initialize(root, filesystem.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := derived.Append(ctx, "ocr/run.txt", []byte("two\n")); !errors.Is(err, filesystem.ErrNotThisVault) {
		t.Errorf("appending into another vault at the same path gave %v", err)
	}
}

func TestWhatIsWrittenIsRead(t *testing.T) {
	derived, root := store(t)
	ctx := t.Context()

	if err := derived.Write(ctx, "ocr/abc.txt", []byte("first")); err != nil {
		t.Fatal(err)
	}
	if got, err := derived.Read(ctx, "ocr/abc.txt"); err != nil || string(got) != "first" {
		t.Fatalf("read %q, %v", got, err)
	}

	// Writing again replaces: one recognition run twice is the same bytes.
	if err := derived.Write(ctx, "ocr/abc.txt", []byte("second")); err != nil {
		t.Fatal(err)
	}
	if got, _ := derived.Read(ctx, "ocr/abc.txt"); string(got) != "second" {
		t.Errorf("read %q after replacing", got)
	}

	if err := derived.Append(ctx, "ocr/abc.txt", []byte(" and more")); err != nil {
		t.Fatal(err)
	}
	if got, _ := derived.Read(ctx, "ocr/abc.txt"); string(got) != "second and more" {
		t.Errorf("read %q after appending", got)
	}

	at := filepath.Join(root, filesystem.DefaultServiceDir, filesystem.OCRDir, "abc.txt")
	if _, err := os.Stat(at); err != nil {
		t.Errorf("the file is not where the store says it is: %v", err)
	}
}

func TestAppendingToNothingBeginsIt(t *testing.T) {
	derived, _ := store(t)

	if err := derived.Append(t.Context(), "ocr/part.txt", []byte("one")); err != nil {
		t.Fatal(err)
	}
	if got, _ := derived.Read(t.Context(), "ocr/part.txt"); string(got) != "one" {
		t.Errorf("read %q", got)
	}
}

func TestANameWithNothingUnderItIsAnAnswer(t *testing.T) {
	// The folder is on the person's disk and they may empty it. That is a thing
	// that happened, not a failure of the application.
	derived, _ := store(t)

	if _, err := derived.Read(t.Context(), "ocr/gone.txt"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("reading nothing gave %v, want fs.ErrNotExist", err)
	}
	if err := derived.Remove(t.Context(), "ocr/gone.txt"); err != nil {
		t.Errorf("removing nothing gave %v, want it to be the outcome asked for", err)
	}
}

func TestTheStoreCannotNameTheVaultsIdentity(t *testing.T) {
	// Every name a store takes begins with its own area, and the vault's
	// identity is in the folder and in no area. So the file every index row
	// points at is not a name this can express: not by naming it, and not by
	// climbing out of the area to reach it.
	derived, root := store(t)
	identity := filepath.Join(root, filesystem.DefaultServiceDir, "config.json")
	was, err := os.ReadFile(identity)
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		"ocr/../config.json",
		"ocr/../../.numen/config.json",
		"config.json",
	} {
		if err := derived.Write(t.Context(), name, []byte("overwritten")); err == nil {
			t.Errorf("the store wrote through %q", name)
		}
	}

	if got, err := os.ReadFile(identity); err != nil || string(got) != string(was) {
		t.Errorf("the vault's identity is now %q, %v", got, err)
	}
}

func TestTheStoreDoesNotWriteThroughALinkOutOfItself(t *testing.T) {
	// A name in the store may be a link that a person, a sync client or another
	// tool put there. Following it would read a note back as a recognition, or
	// write a recognition over a note.
	derived, root := store(t)
	note := filepath.Join(root, "Note.md")
	if err := os.WriteFile(note, []byte("# Note\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	area := filepath.Join(root, filesystem.DefaultServiceDir, filesystem.OCRDir)
	if err := os.MkdirAll(area, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(note, filepath.Join(area, "linked.txt")); err != nil {
		t.Skipf("this filesystem has no links: %v", err)
	}

	if err := derived.Write(t.Context(), "ocr/linked.txt", []byte("overwritten")); err == nil {
		t.Error("the store wrote through a link out of itself")
	}
	if _, err := derived.Read(t.Context(), "ocr/linked.txt"); err == nil {
		t.Error("the store read through a link out of itself")
	}
	if got, _ := os.ReadFile(note); string(got) != "# Note\n" {
		t.Errorf("the note is now %q", got)
	}
}

func TestAStoreIsOneFolder(t *testing.T) {
	root := t.TempDir()
	for _, area := range []string{"a/b", "..", ".", `a\b`} {
		if _, err := filesystem.OpenDerived(root, filesystem.Options{}, area); err == nil {
			t.Errorf("opened a store on %q", area)
		}
	}
}

func TestANameIsClaimedByOneCallerAtATime(t *testing.T) {
	// One recognition writes one file, appending to it for an hour. The name is
	// held for as long as that takes.
	derived, _ := store(t)

	release, err := derived.Claim(t.Context(), "ocr/abc.txt")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := derived.Claim(t.Context(), "ocr/abc.txt"); !errors.Is(err, port.ErrClaimed) {
		t.Errorf("claiming a held name gave %v, want port.ErrClaimed", err)
	}

	if err := release(); err != nil {
		t.Fatal(err)
	}
	again, err := derived.Claim(t.Context(), "ocr/abc.txt")
	if err != nil {
		t.Fatalf("claiming a released name gave %v", err)
	}
	if err := again(); err != nil {
		t.Error(err)
	}
}

func TestTwoNamesAreClaimedTogether(t *testing.T) {
	derived, _ := store(t)

	first, err := derived.Claim(t.Context(), "ocr/one.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer first()
	second, err := derived.Claim(t.Context(), "ocr/two.txt")
	if err != nil {
		t.Fatalf("claiming another name gave %v", err)
	}
	if err := second(); err != nil {
		t.Error(err)
	}
}

func TestAClaimCannotLeaveTheStore(t *testing.T) {
	// A claim is a file the store writes, and it is judged by the rule every
	// other name is judged by.
	derived, _ := store(t)

	for _, name := range []string{
		"ocr/../config.json",
		"ocr/../../elsewhere",
		"config.json",
		"/etc/passwd",
	} {
		if _, err := derived.Claim(t.Context(), name); !errors.Is(err, filesystem.ErrOutside) {
			t.Errorf("claiming %q gave %v, want ErrOutside", name, err)
		}
	}
}

// claimHeld is what the child process prints once it holds the claim, and
// holdingEnv is the vault it is asked to hold it in.
const (
	claimHeld  = "the claim is held"
	holdingEnv = "NUMEN_TEST_CLAIM_ROOT"
)

func TestAClaimHeldByAnotherProcessIsRefused(t *testing.T) {
	// `numen recognise` is a separate binary and runs while the window is open,
	// so what one process holds every other has to see.
	derived, root := store(t)

	child := exec.CommandContext(t.Context(), os.Args[0],
		"-test.run=^TestHoldingAClaimForAnotherProcess$", "-test.timeout=1m")
	child.Env = append(os.Environ(), holdingEnv+"="+root)
	child.Stderr = os.Stderr
	stdin, err := child.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		stdin.Close()
		child.Wait()
	}()

	said := bufio.NewScanner(stdout)
	for said.Scan() && said.Text() != claimHeld {
	}
	if said.Text() != claimHeld {
		t.Fatalf("the child never took the claim: %v", said.Err())
	}

	if _, err := derived.Claim(t.Context(), "ocr/held.txt"); !errors.Is(err, port.ErrClaimed) {
		t.Errorf("claiming a name another process holds gave %v, want port.ErrClaimed", err)
	}
}

// TestHoldingAClaimForAnotherProcess is the child of the test above. It takes
// the claim, says so, and holds it until its parent closes its input.
func TestHoldingAClaimForAnotherProcess(t *testing.T) {
	root := os.Getenv(holdingEnv)
	if root == "" {
		t.Skip("this test is run by TestAClaimHeldByAnotherProcessIsRefused")
	}
	derived, err := filesystem.OpenDerived(root, filesystem.Options{}, filesystem.OCRDir)
	if err != nil {
		t.Fatal(err)
	}
	release, err := derived.Claim(t.Context(), "ocr/held.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	fmt.Println(claimHeld)
	io.ReadAll(os.Stdin)
}

// A store says what names it holds, because work whose whole record is a folder
// of files nobody names in advance has no other way to find them.
func TestTheStoreSaysWhatNamesItHolds(t *testing.T) {
	derived, _ := store(t)
	ctx := t.Context()

	if got, err := derived.List(ctx, filesystem.OCRDir); err != nil || len(got) != 0 {
		t.Fatalf("an empty store lists %v, %v", got, err)
	}

	for _, name := range []string{"ocr/second.txt", "ocr/first.txt"} {
		if err := derived.Write(ctx, name, []byte("in it")); err != nil {
			t.Fatal(err)
		}
	}
	if err := derived.Write(ctx, "ocr/under/deeper.txt", []byte("below")); err != nil {
		t.Fatal(err)
	}

	got, err := derived.List(ctx, filesystem.OCRDir)
	if err != nil {
		t.Fatal(err)
	}
	want := []port.Stored{
		{Name: "ocr/first.txt", Size: len("in it")},
		{Name: "ocr/second.txt", Size: len("in it")},
	}
	if !slices.Equal(got, want) {
		t.Errorf("listed %v, want the two files it holds with their lengths, sorted", got)
	}
}

// What a listing says about a file is how long it now is, so a reader that has
// read it once is told when it has grown.
func TestAListingSaysHowLongEachFileIs(t *testing.T) {
	derived, _ := store(t)
	ctx := t.Context()

	if err := derived.Write(ctx, "ocr/one.txt", []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := derived.Append(ctx, "ocr/one.txt", []byte(" and more")); err != nil {
		t.Fatal(err)
	}

	got, err := derived.List(ctx, filesystem.OCRDir)
	if err != nil {
		t.Fatal(err)
	}
	want := []port.Stored{{Name: "ocr/one.txt", Size: len("first and more")}}
	if !slices.Equal(got, want) {
		t.Errorf("listed %v, want %v", got, want)
	}
}

// A name that leaves the store is refused here as it is everywhere else.
func TestTheStoreListsNothingOutsideItself(t *testing.T) {
	derived, _ := store(t)
	if _, err := derived.List(t.Context(), "ocr/../.."); err == nil {
		t.Error("the store listed a folder outside itself")
	}
}
