package derivedfile_test

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

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// testArea is the one folder of the service folder the stores of these tests answer for.
const testArea = "area"

func store(t *testing.T) (*derivedfile.DerivedStore, string) {
	t.Helper()
	root := t.TempDir()
	if _, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}
	derived, err := derivedfile.OpenDerived(root, vaultconfig.Options{}, testArea)
	if err != nil {
		t.Fatal(err)
	}
	return derived, root
}

func TestOpeningTheStoreWritesNothing(t *testing.T) {
	// The application opens every vault it holds. Opening a store must not
	// create a folder in any of them.
	derived, root := store(t)
	_ = derived

	pos := filepath.Join(root, vaultconfig.DefaultServiceDir, testArea)
	if _, err := os.Stat(pos); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("opening the store made %s: %v", testArea, err)
	}
}

// A vault folder taken away under a store that is writing is not a vault to go
// on writing into: creating the parents of a name would put a stub where the
// vault was, and what is written there is shadowed the moment the real one is
// back.
func TestAppend_RejectsMissingVaultFolder(t *testing.T) {
	derived, root := store(t)
	ctx := t.Context()
	if err := derived.Append(ctx, "area/run.txt", []byte("one\n")); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}

	if err := derived.Append(ctx, "area/run.txt", []byte("two\n")); !errors.Is(err, derivedfile.ErrNotThisVault) {
		t.Errorf("appending into a vault that is gone gave %v", err)
	}
	if err := derived.Write(ctx, "area/run.txt", []byte("two\n")); !errors.Is(err, derivedfile.ErrNotThisVault) {
		t.Errorf("writing into a vault that is gone gave %v", err)
	}
	if _, err := derived.List(ctx, testArea); !errors.Is(err, derivedfile.ErrNotThisVault) {
		t.Errorf("listing a vault that is gone gave %v", err)
	}
	if _, err := os.Stat(root); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the vault folder was made again to write into: %v", err)
	}

	// A folder put back at the path, carrying an identity of its own, is
	// another vault and not this one.
	if _, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := derived.Append(ctx, "area/run.txt", []byte("two\n")); !errors.Is(err, derivedfile.ErrNotThisVault) {
		t.Errorf("appending into another vault at the same path gave %v", err)
	}
}

// A store opened on a folder without an identity has none to lose, so the
// folder itself is the evidence that it still exists.
func TestAppend_RejectsMissingFolderWithoutIdentity(t *testing.T) {
	root := filepath.Join(t.TempDir(), "vault")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	derived, err := derivedfile.OpenDerived(root, vaultconfig.Options{}, testArea)
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	if err := derived.Append(ctx, "area/run.txt", []byte("one\n")); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	if err := derived.Append(ctx, "area/run.txt", []byte("two\n")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("appending into a folder that is gone gave %v", err)
	}
	if _, err := os.Stat(root); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the folder was made again to write into: %v", err)
	}
}

// The identity is read out of a file, and a store that has read it once goes on
// answering from that recognition. A file written again is another recognition.
func TestAStoreNoticesTheIdentityChangingBelowIt(t *testing.T) {
	derived, root := store(t)
	ctx := t.Context()
	if err := derived.Append(ctx, "area/run.txt", []byte("one\n")); err != nil {
		t.Fatal(err)
	}

	// A synchroniser bringing another machine's vault down over this path
	// leaves the folder where it was and the identity in it another.
	pos := filepath.Join(root, vaultconfig.DefaultServiceDir, "config.json")
	other := `{"v":1,"id":"01J0000000000000000000000A"}` + "\n"
	if err := os.WriteFile(pos, []byte(other), 0o644); err != nil {
		t.Fatal(err)
	}
	when := time.Now().Add(time.Second)
	if err := os.Chtimes(pos, when, when); err != nil {
		t.Fatal(err)
	}

	if err := derived.Append(ctx, "area/run.txt", []byte("two\n")); !errors.Is(err, derivedfile.ErrNotThisVault) {
		t.Errorf("appending into another vault at the same path gave %v", err)
	}
}

func TestWhatIsSavedIsRead(t *testing.T) {
	derived, root := store(t)
	ctx := t.Context()

	if err := derived.Write(ctx, "area/abc.txt", []byte("first")); err != nil {
		t.Fatal(err)
	}
	if got, err := derived.Read(ctx, "area/abc.txt"); err != nil || string(got) != "first" {
		t.Fatalf("read %q, %v", got, err)
	}

	// Writing again replaces: one recognition run twice is the same bytes.
	if err := derived.Write(ctx, "area/abc.txt", []byte("second")); err != nil {
		t.Fatal(err)
	}
	if got, _ := derived.Read(ctx, "area/abc.txt"); string(got) != "second" {
		t.Errorf("read %q after replacing", got)
	}

	if err := derived.Append(ctx, "area/abc.txt", []byte(" and more")); err != nil {
		t.Fatal(err)
	}
	if got, _ := derived.Read(ctx, "area/abc.txt"); string(got) != "second and more" {
		t.Errorf("read %q after appending", got)
	}

	pos := filepath.Join(root, vaultconfig.DefaultServiceDir, testArea, "abc.txt")
	if _, err := os.Stat(pos); err != nil {
		t.Errorf("the file is not where the store says it is: %v", err)
	}
}

func TestAppendingToNothingBeginsIt(t *testing.T) {
	derived, _ := store(t)

	if err := derived.Append(t.Context(), "area/part.txt", []byte("one")); err != nil {
		t.Fatal(err)
	}
	if got, _ := derived.Read(t.Context(), "area/part.txt"); string(got) != "one" {
		t.Errorf("read %q", got)
	}
}

func TestANameWithNothingBelowItIsAnAnswer(t *testing.T) {
	// The folder is on the user's disk and can be emptied. A missing name is an
	// expected state, not an application failure.
	derived, _ := store(t)

	if _, err := derived.Read(t.Context(), "area/gone.txt"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("reading nothing gave %v, want fs.ErrNotExist", err)
	}
	if err := derived.Remove(t.Context(), "area/gone.txt"); err != nil {
		t.Errorf("removing nothing gave %v, want it to be the outcome asked for", err)
	}
}

func TestTheStoreCannotNameTheVaultsIdentity(t *testing.T) {
	// Every name a store takes begins with its own area, and the vault's
	// identity is in the folder and in no area. So the file every index row
	// points at is not a name this can express: not by naming it, and not by
	// climbing out of the area to reach it.
	derived, root := store(t)
	identity := filepath.Join(root, vaultconfig.DefaultServiceDir, "config.json")
	was, err := os.ReadFile(identity)
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		"area/../config.json",
		"area/../../.numen/config.json",
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
	// A name in the store may be a link created by the user, a sync client or
	// another tool. Following it would read a note back as a recognition, or
	// write a recognition over a note.
	derived, root := store(t)
	note := filepath.Join(root, "Note.md")
	if err := os.WriteFile(note, []byte("# Note\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	area := filepath.Join(root, vaultconfig.DefaultServiceDir, testArea)
	if err := os.MkdirAll(area, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(note, filepath.Join(area, "linked.txt")); err != nil {
		t.Skipf("this filesystem has no links: %v", err)
	}

	if err := derived.Write(t.Context(), "area/linked.txt", []byte("overwritten")); err == nil {
		t.Error("the store wrote through a link out of itself")
	}
	if _, err := derived.Read(t.Context(), "area/linked.txt"); err == nil {
		t.Error("the store read through a link out of itself")
	}
	if got, _ := os.ReadFile(note); string(got) != "# Note\n" {
		t.Errorf("the note is now %q", got)
	}
}

func TestTheStoreDoesNotWriteThroughALinkOutOfItsArea(t *testing.T) {
	// The vault's identity is in the service folder and in no area, so no name
	// addresses it. A link inside an area can still point to it, and appending a
	// transcript through that link would overwrite the identity.
	derived, root := store(t)
	identity := filepath.Join(root, vaultconfig.DefaultServiceDir, "config.json")
	before, err := os.ReadFile(identity)
	if err != nil {
		t.Fatal(err)
	}
	area := filepath.Join(root, vaultconfig.DefaultServiceDir, testArea)
	if err := os.MkdirAll(area, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "config.json"), filepath.Join(area, "run.txt")); err != nil {
		t.Skipf("this filesystem has no links: %v", err)
	}

	if err := derived.Append(t.Context(), "area/run.txt", []byte("transcript\n")); err == nil {
		t.Error("the store appended through a link out of its area")
	}
	if err := derived.Write(t.Context(), "area/run.txt", []byte("transcript\n")); err == nil {
		t.Error("the store wrote through a link out of its area")
	}
	if _, err := derived.Read(t.Context(), "area/run.txt"); err == nil {
		t.Error("the store read the vault's identity back as a recognition")
	}
	if got, _ := os.ReadFile(identity); string(got) != string(before) {
		t.Errorf("the vault's identity is now %q", got)
	}
}

func TestAStoreIsOneFolder(t *testing.T) {
	root := t.TempDir()
	for _, area := range []string{"a/b", "..", ".", `a\b`} {
		if _, err := derivedfile.OpenDerived(root, vaultconfig.Options{}, area); err == nil {
			t.Errorf("opened a store on %q", area)
		}
	}
}

func TestANameIsClaimedByOneCallerAtATime(t *testing.T) {
	// One recognition writes one file, appending to it for an hour. The name is
	// held for as long as that takes.
	derived, _ := store(t)

	release, err := derived.ClaimName(t.Context(), "area/abc.txt")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := derived.ClaimName(t.Context(), "area/abc.txt"); !errors.Is(err, vaultport.ErrClaimed) {
		t.Errorf("claiming a held name gave %v, want port.ErrClaimed", err)
	}

	if err := release(); err != nil {
		t.Fatal(err)
	}
	again, err := derived.ClaimName(t.Context(), "area/abc.txt")
	if err != nil {
		t.Fatalf("claiming a released name gave %v", err)
	}
	if err := again(); err != nil {
		t.Error(err)
	}
}

func TestTwoNamesAreClaimedTogether(t *testing.T) {
	derived, _ := store(t)

	first, err := derived.ClaimName(t.Context(), "area/one.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer first()
	second, err := derived.ClaimName(t.Context(), "area/two.txt")
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
		"area/../config.json",
		"area/../../elsewhere",
		"config.json",
		"/etc/passwd",
	} {
		if _, err := derived.ClaimName(t.Context(), name); !errors.Is(err, vaultconfig.ErrOutside) {
			t.Errorf("claiming %q gave %v, want ErrOutside", name, err)
		}
	}
}

// claimLocked is the line the child process prints once it holds the claim, and
// holdingEnv is the environment variable naming the vault it claims.
const (
	claimLocked = "the claim is held"
	holdingEnv  = "NUMEN_TEST_CLAIM_ROOT"
)

func TestClaimName_RejectsClaimFromOtherProcess(t *testing.T) {
	// `numen recognise` is a separate binary and runs while the window is open,
	// so what one process holds every other has to see.
	derived, root := store(t)

	child := exec.CommandContext(t.Context(), os.Args[0],
		"-test.run=^TestContainingAClaimForAnotherProcess$", "-test.timeout=1m")
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

	// The child may print other lines first; the loop waits for the claim line.
	response := bufio.NewScanner(stdout)
	for response.Scan() {
		if response.Text() == claimLocked {
			break
		}
	}
	if response.Text() != claimLocked {
		t.Fatalf("the child never took the claim: %v", response.Err())
	}

	if _, err := derived.ClaimName(t.Context(), "area/held.txt"); !errors.Is(err, vaultport.ErrClaimed) {
		t.Errorf("claiming a name another process holds gave %v, want port.ErrClaimed", err)
	}
}

// TestContainingAClaimForAnotherProcess is the child of the test above. It
// takes the claim, prints the claim line, and holds the claim until its parent
// closes its input.
func TestContainingAClaimForAnotherProcess(t *testing.T) {
	root := os.Getenv(holdingEnv)
	if root == "" {
		t.Skip("this test is run by TestClaimName_RejectsClaimFromOtherProcess")
	}
	derived, err := derivedfile.OpenDerived(root, vaultconfig.Options{}, testArea)
	if err != nil {
		t.Fatal(err)
	}
	release, err := derived.ClaimName(t.Context(), "area/held.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	fmt.Println(claimLocked)
	io.ReadAll(os.Stdin)
}

// A store lists the names it holds, because a record that is a folder of files
// with unpredictable names can only be discovered by listing.
func TestTheStoreReturnsWhatNamesItContains(t *testing.T) {
	derived, _ := store(t)
	ctx := t.Context()

	if got, err := derived.List(ctx, testArea); err != nil || len(got) != 0 {
		t.Fatalf("an empty store lists %v, %v", got, err)
	}

	for _, name := range []string{"area/second.txt", "area/first.txt"} {
		if err := derived.Write(ctx, name, []byte("in it")); err != nil {
			t.Fatal(err)
		}
	}
	if err := derived.Write(ctx, "area/under/deeper.txt", []byte("below")); err != nil {
		t.Fatal(err)
	}

	got, err := derived.List(ctx, testArea)
	if err != nil {
		t.Fatal(err)
	}
	want := []derivedfile.Listed{
		{Name: "area/first.txt", Size: len("in it")},
		{Name: "area/second.txt", Size: len("in it")},
	}
	if !slices.Equal(got, want) {
		t.Errorf("listed %v, want the two files it holds with their lengths, sorted", got)
	}
}

// A listing reports the current length of each file, so a reader that has
// already read a file can detect that it has grown.
func TestAListingReturnsHowLongEachFileIs(t *testing.T) {
	derived, _ := store(t)
	ctx := t.Context()

	if err := derived.Write(ctx, "area/one.txt", []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := derived.Append(ctx, "area/one.txt", []byte(" and more")); err != nil {
		t.Fatal(err)
	}

	got, err := derived.List(ctx, testArea)
	if err != nil {
		t.Fatal(err)
	}
	want := []derivedfile.Listed{{Name: "area/one.txt", Size: len("first and more")}}
	if !slices.Equal(got, want) {
		t.Errorf("listed %v, want %v", got, want)
	}
}

func TestTheStoreListsNothingOutsideItself(t *testing.T) {
	derived, _ := store(t)
	if _, err := derived.List(t.Context(), "area/../.."); err == nil {
		t.Error("the store listed a folder outside itself")
	}
}

// A use case reading what two producers wrote opens one store on both areas.
func TestAStoreAnswersForEveryAreaItWasOpenedOn(t *testing.T) {
	root := t.TempDir()
	if _, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}
	derived, err := derivedfile.OpenDerived(root, vaultconfig.Options{}, testArea, "other")
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()

	for _, name := range []string{"area/abc.txt", "other/abc.vtt"} {
		if err := derived.Write(ctx, name, []byte("said")); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if err := derived.Write(ctx, "elsewhere/abc.jsonl", []byte("answered")); err == nil {
		t.Error("the store wrote into an area it was not opened on")
	}
	if area := derived.Area(); area != testArea {
		t.Errorf("the store is called %q, want the first area it was opened on", area)
	}
}

// The store's own folder is resolved once. A folder that replaces it afterwards
// (a sync client keeping the application's files on another disk, or the user
// moving them) becomes the location the store writes to, so the resolved
// folder is the current one.
func TestAStoreFollowsItsFolderWhereItIsInsertAfterwards(t *testing.T) {
	target := t.TempDir()
	root := filepath.Join(t.TempDir(), "vault")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	derived, err := derivedfile.OpenDerived(root, vaultconfig.Options{}, testArea)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, vaultconfig.DefaultServiceDir)); err != nil {
		t.Skipf("this filesystem has no links: %v", err)
	}

	ctx := t.Context()
	if err := derived.Write(ctx, "area/abc.txt", []byte("read")); err != nil {
		t.Fatalf("writing into the folder the store now has: %v", err)
	}
	got, err := derived.Read(ctx, "area/abc.txt")
	if err != nil || string(got) != "read" {
		t.Errorf("read %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(target, testArea, "abc.txt")); err != nil {
		t.Errorf("the file did not land where the folder is: %v", err)
	}
}

// the rest of it.
func TestEveryAreaOfAStoreIsOneFolder(t *testing.T) {
	root := t.TempDir()
	if _, err := derivedfile.OpenDerived(root, vaultconfig.Options{}, testArea, "a/b"); err == nil {
		t.Error("opened a store on a name that is not one folder")
	}
}

// An area is one folder of the store's own. A link standing where the area
// should be — here to the service folder itself — is not one, and a name under
// it lands nowhere: the vault's identity is a step down that link, and a store
// that followed it would write over the one file every index row points at.
func TestAnAreaThatIsALinkContainsNothing(t *testing.T) {
	derived, root := store(t)
	identity := filepath.Join(root, vaultconfig.DefaultServiceDir, "config.json")
	before, err := os.ReadFile(identity)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(".", filepath.Join(root, vaultconfig.DefaultServiceDir, testArea)); err != nil {
		t.Skipf("this filesystem has no links: %v", err)
	}

	ctx := t.Context()
	if _, err := derived.Read(ctx, "area/config.json"); err == nil {
		t.Error("the store read the vault's identity through an area that is a link")
	}
	if err := derived.Write(ctx, "area/config.json", []byte(`{"id":"stolen"}`)); err == nil {
		t.Error("the store wrote the vault's identity through an area that is a link")
	}
	if err := derived.Append(ctx, "area/config.json", []byte("transcript\n")); err == nil {
		t.Error("the store appended to the vault's identity through an area that is a link")
	}
	if _, err := derived.List(ctx, testArea); err == nil {
		t.Error("the store listed the service folder as its area")
	}
	if got, _ := os.ReadFile(identity); string(got) != string(before) {
		t.Errorf("the vault's identity is now %q", got)
	}
}

// The store's folder is resolved once, and the result holds only while the
// folder is unchanged. Here the folder is repointed from a subfolder to its
// parent, its identity moves with it as the same file, and a link inside the new
// folder leads back into the old one: a name through that link lands in the old
// area, which is no longer an area of the store, and the store opened before the
// move rejects it exactly as one opened after the move would.
func TestAStoreDoesNotWriteIntoWhereItsFolderWasBefore(t *testing.T) {
	target := t.TempDir()
	old := filepath.Join(target, "old")
	root := filepath.Join(t.TempDir(), "vault")
	for _, folder := range []string{old, root, filepath.Join(target, testArea)} {
		if err := os.MkdirAll(folder, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	service := filepath.Join(root, vaultconfig.DefaultServiceDir)
	if err := os.Symlink(old, service); err != nil {
		t.Skipf("this filesystem has no links: %v", err)
	}
	if _, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}
	derived, err := derivedfile.OpenDerived(root, vaultconfig.Options{}, testArea)
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	if err := derived.Write(ctx, "area/a.txt", []byte("one")); err != nil {
		t.Fatalf("writing into the folder the store has: %v", err)
	}

	if err := os.Rename(filepath.Join(old, "config.json"), filepath.Join(target, "config.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(service); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, service); err != nil {
		t.Fatal(err)
	}
	back := filepath.Join(target, testArea, "back")
	if err := os.Symlink(filepath.Join("..", "old", testArea), back); err != nil {
		t.Fatal(err)
	}

	if err := derived.Write(ctx, "area/back/x.txt", []byte("two")); err == nil {
		t.Error("the store wrote into where its folder was before")
	}
	if _, err := os.Stat(filepath.Join(old, testArea, "x.txt")); err == nil {
		t.Error("the file landed in the old area")
	}
	fresh, err := derivedfile.OpenDerived(root, vaultconfig.Options{}, testArea)
	if err != nil {
		t.Fatal(err)
	}
	if err := fresh.Write(ctx, "area/back/y.txt", []byte("three")); err == nil {
		t.Error("a store opened after the move wrote into the old area")
	}
	if err := derived.Write(ctx, "area/b.txt", []byte("four")); err != nil {
		t.Errorf("writing into the folder the store now has: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, testArea, "b.txt")); err != nil {
		t.Errorf("the file did not land where the folder is: %v", err)
	}
}
