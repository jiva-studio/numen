package check_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/check"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/testsupport"
)

// checked scans a vault and hands back the checks over it.
func checked(t *testing.T, notes map[string]string) (check.Checks, domain.Vault) {
	t.Helper()
	v := testsupport.NewVault(t, notes)
	db, err := container.Config{
		IndexPath: filepath.Join(t.TempDir(), "index.db"),
	}.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	scan := usecase.Scan{
		Readers: filesystem.Readers{}, Vaults: db.Vaults(), Notes: db.Notes(),
		Known: db.Queries(), Maintenance: db.Maintenance(),
	}
	if _, err := scan.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	return check.Standard(db.Problems()), v
}

func run(t *testing.T, l check.Checks, v domain.Vault, named ...domain.Check) []domain.VaultProblem {
	t.Helper()
	found, err := l.Run(t.Context(), v, named...)
	if err != nil {
		t.Fatal(err)
	}
	return found
}

func only(found []domain.VaultProblem, check domain.Check) []domain.VaultProblem {
	var out []domain.VaultProblem
	for _, p := range found {
		if p.Check == check {
			out = append(out, p)
		}
	}
	return out
}

// The point of the whole arrangement: the problem is against the note somebody
// opens to settle it, which for an ambiguous link is the note that wrote it.
func TestAnAmbiguousLinkIsTheProblemOfTheNoteThatWroteIt(t *testing.T) {
	l, v := checked(t, map[string]string{
		"physics/Entropy.md":   "# Entropy\n",
		"chemistry/Entropy.md": "# Entropy\n",
		"Heat.md":              "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n# Heat\n",
	})

	found := only(run(t, l, v), domain.CheckAmbiguous)
	if len(found) != 1 {
		t.Fatalf("want one ambiguous link, got %+v", found)
	}
	if found[0].Path != "Heat.md" {
		t.Errorf("want it against the note that wrote the link, got %s", found[0].Path)
	}
	if len(found[0].Candidates) != 2 {
		t.Errorf("want both notes it could mean, got %v", found[0].Candidates)
	}
	if !strings.Contains(found[0].Detail, "Entropy") {
		t.Errorf("the detail names nothing useful: %q", found[0].Detail)
	}
}

// A name that is an exact path is not ambiguous, however many notes share the
// filename. Reporting it would be the check disagreeing with the resolver.
func TestANameThatResolvesExactlyIsNotAmbiguous(t *testing.T) {
	l, v := checked(t, map[string]string{
		"Entropy.md":         "# Entropy\n",
		"physics/Entropy.md": "# Entropy\n",
		"Heat.md":            "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n# Heat\n",
	})

	if found := only(run(t, l, v), domain.CheckAmbiguous); len(found) != 0 {
		t.Errorf("the link reaches one note exactly: %+v", found)
	}
}

// Quiet: a vault being written is full of links to notes not yet made, and they
// would bury everything else.
func TestADanglingLinkArrivesOnlyWhenAskedFor(t *testing.T) {
	l, v := checked(t, map[string]string{
		"Heat.md": "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n# Heat\n",
	})

	if found := only(run(t, l, v), domain.CheckDangling); len(found) != 0 {
		t.Errorf("dangling should be quiet by default: %+v", found)
	}

	found := run(t, l, v, domain.CheckDangling)
	if len(found) != 1 || found[0].Path != "Heat.md" {
		t.Fatalf("want the note that wrote it, got %+v", found)
	}
	if got := found[0].Target.String(); got != "name://Entropy" {
		t.Errorf("want what the link says, got %q", got)
	}
}

// Writing the missing note mends it, and nobody touches the link: what a name
// reaches is worked out when it is asked.
func TestWritingTheMissingNoteMendsADanglingLink(t *testing.T) {
	l, v := checked(t, map[string]string{
		"Heat.md":    "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n# Heat\n",
		"Entropy.md": "# Entropy\n",
	})

	if found := run(t, l, v, domain.CheckDangling); len(found) != 0 {
		t.Errorf("nothing is dangling here: %+v", found)
	}
}

func TestWhatOneFileGotWrongIsReportedAgainstThatFile(t *testing.T) {
	l, v := checked(t, map[string]string{
		"Heat.md":    "---\nlinks:\n  - to: Entropy\n---\n# Heat\n",
		"Broken.md":  "---\nid: [unterminated\n---\n# Broken\n",
		"Entropy.md": "# Entropy\n",
	})
	found := run(t, l, v)

	parsed := only(found, domain.CheckParse)
	if len(parsed) != 1 || parsed[0].Path != "Heat.md" {
		t.Errorf("want the link with no role against Heat.md, got %+v", parsed)
	}
	unreadable := only(found, domain.CheckFrontmatter)
	if len(unreadable) != 1 || unreadable[0].Path != "Broken.md" {
		t.Fatalf("want the unreadable block against Broken.md, got %+v", unreadable)
	}
	if !strings.Contains(unreadable[0].Detail, "nothing may be written") {
		t.Errorf("the detail should say what it costs: %q", unreadable[0].Detail)
	}
}

func TestAskingForACheckThatDoesNotExistSaysWhatDoes(t *testing.T) {
	l, v := checked(t, nil)

	_, err := l.Run(t.Context(), v, "spelling")
	if err == nil {
		t.Fatal("want an error")
	}
	for _, name := range []string{"parse", "frontmatter", "ambiguous", "dangling"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("the error should list %s: %v", name, err)
		}
	}
}

// Adding a check is adding a file, and what runs is whatever the set holds.
func TestTheSetOfChecksIsWhatRuns(t *testing.T) {
	l, v := checked(t, map[string]string{"Heat.md": "---\nlinks:\n  - to: Entropy\n---\n# Heat\n"})
	l.Checks = nil

	if found := run(t, l, v); len(found) != 0 {
		t.Errorf("an empty set of checks found %+v", found)
	}
}

// An identifier no vault here holds is not a broken link: it names one note in
// the world, and the vault holding it may not be open on this machine. The
// resolver says so deliberately, and the check must not disagree with it.
func TestAnIdentifierFromAnotherVaultIsNotDangling(t *testing.T) {
	l, v := checked(t, map[string]string{
		"Heat.md": "---\nlinks:\n  - to: note://01J8F3K2M9QRSTVWXYZ012\n    role: parent\n---\n# Heat\n",
	})

	if found := run(t, l, v, domain.CheckDangling); len(found) != 0 {
		t.Errorf("a link to another vault was called broken: %+v", found)
	}
}
