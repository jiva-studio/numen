package filesystem

import (
	"path/filepath"
	"runtime"
	"testing"
)

// paths is every shape a path can arrive in, whether or not any file of that
// name exists. The rules are about the text of a path, and the two of them
// have to answer every one of these between them.
var paths = []string{
	"note.md",
	"notes/Entropy.md",
	"assets/paper.pdf",
	".obsidian/note.md",
	".numen",
	".numen/config.json",
	".numen/ocr/abc.txt",
	".numen/../notes/a.md",
	"notes/../.numen/config.json",
	".NUMEN/config.json",
	"NUL",
	"c:notes.md",
	"notes/../../etc/passwd",
	"..",
	".",
	"../outside.md",
	"/absolute.md",
	"",
	"with\x00nul.md",
	"./note.md",
	"a//b.md",
}

// TestOneOfTheTwoRulesAnswers is the property the pair exists for: a path is
// the vault's or the application's and never both. Were they to overlap, a
// writer of notes could be handed a path into the application's own folder, and
// the refusal that keeps an artifact from being read back as a note would hold
// only for the call sites that happen to exist today.
func TestOneOfTheTwoRulesAnswers(t *testing.T) {
	root := t.TempDir()
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			isVault, _, vaultErr := resolveVaultPath(resolved, path, DefaultServiceDir)
			isOurs, _, oursErr := service(resolved, path, DefaultServiceDir)

			switch {
			case vaultErr == nil && oursErr == nil:
				t.Errorf("%q is both the vault's (%s) and the application's (%s)", path, isVault, isOurs)
			case vaultErr != nil && oursErr != nil:
				// A path that is neither is an ordinary outcome: it leaves the
				// vault, or it names nothing.
			}
		})
	}
}

// A link is a second spelling for a place, and the rules are asked about the
// place. One pointing at the application's folder is the application's however
// it is spelled; one pointing at a folder of notes is still the vault's, which
// is the arrangement a person makes on purpose and which nothing here narrows.
func TestALinkIsJudgedByWhereItLeads(t *testing.T) {
	root := vault(t)

	tests := []struct {
		name    string
		path    string
		isVault bool
		isOurs  bool
	}{
		{name: "a link to notes", path: "inward/Entropy.md", isVault: true},
		{name: "into the application's folder", path: "held/ocr/abc.txt"},
		{name: "the link to it", path: "held"},
		{name: "the application's folder itself", path: ".numen/ocr/abc.txt", isOurs: true},
		{name: "out of the vault", path: "outward/Entropy.md"},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			if _, _, err := resolveVaultPath(root, c.path, DefaultServiceDir); (err == nil) != c.isVault {
				t.Errorf("resolveVaultPath(%q) gave %v, want accepted=%v", c.path, err, c.isVault)
			}
			if _, _, err := service(root, c.path, DefaultServiceDir); (err == nil) != c.isOurs {
				t.Errorf("service(%q) gave %v, want accepted=%v", c.path, err, c.isOurs)
			}
		})
	}
}

// On Windows a name kept for a device and a path relative to a drive name
// nothing a vault holds. On the other systems they are ordinary names.
func TestAWindowsDeviceNameIsNotAPathInTheVault(t *testing.T) {
	root := t.TempDir()
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"NUL", "notes/con.md", "c:notes.md"} {
		_, _, err := resolveVaultPath(resolved, path, DefaultServiceDir)
		if held := err == nil; held == (runtime.GOOS == "windows") {
			t.Errorf("resolveVaultPath(%q) gave %v on %s", path, err, runtime.GOOS)
		}
	}
}

func TestWhatEachRuleAnswersFor(t *testing.T) {
	root := t.TempDir()
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		isVault bool // resolveVaultPath accepts it
		isOurs  bool // service accepts it
	}{
		{name: "a note", path: "notes/Entropy.md", isVault: true},
		{name: "a document", path: "assets/paper.pdf", isVault: true},
		{name: "another tool's folder", path: ".obsidian/note.md", isVault: true},
		{name: "the service folder itself", path: ".numen", isOurs: true},
		{name: "the vault's identity", path: ".numen/config.json", isOurs: true},
		{name: "a derived file", path: ".numen/ocr/abc.txt", isOurs: true},
		{name: "the service folder in capitals", path: ".NUMEN/config.json", isOurs: true},
		{name: "the service folder in either case", path: ".Numen/ocr/abc.txt", isOurs: true},
		{name: "into the service folder the long way", path: "notes/../.numen/config.json", isOurs: true},
		{name: "out of the service folder the long way", path: ".numen/../notes/a.md", isVault: true},
		{name: "out of the vault", path: "notes/../../etc/passwd"},
		{name: "absolute", path: "/etc/passwd"},
		{name: "empty", path: ""},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			if _, _, err := resolveVaultPath(resolved, c.path, DefaultServiceDir); (err == nil) != c.isVault {
				t.Errorf("resolveVaultPath(%q) gave %v, want accepted=%v", c.path, err, c.isVault)
			}
			if _, _, err := service(resolved, c.path, DefaultServiceDir); (err == nil) != c.isOurs {
				t.Errorf("service(%q) gave %v, want accepted=%v", c.path, err, c.isOurs)
			}
		})
	}
}
