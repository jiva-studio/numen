package filesystem

import (
	"path/filepath"
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
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			vault, _, vaultErr := within(real, path, DefaultServiceDir)
			ours, _, oursErr := service(real, path, DefaultServiceDir)

			switch {
			case vaultErr == nil && oursErr == nil:
				t.Errorf("%q is both the vault's (%s) and the application's (%s)", path, vault, ours)
			case vaultErr != nil && oursErr != nil:
				// A path that is neither is an ordinary outcome: it leaves the
				// vault, or it names nothing.
			}
		})
	}
}

func TestWhatEachRuleAnswersFor(t *testing.T) {
	root := t.TempDir()
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		path  string
		vault bool // within accepts it
		ours  bool // service accepts it
	}{
		{name: "a note", path: "notes/Entropy.md", vault: true},
		{name: "a document", path: "assets/paper.pdf", vault: true},
		{name: "another tool's folder", path: ".obsidian/note.md", vault: true},
		{name: "the service folder itself", path: ".numen", ours: true},
		{name: "the vault's identity", path: ".numen/config.json", ours: true},
		{name: "a derived file", path: ".numen/ocr/abc.txt", ours: true},
		{name: "into the service folder the long way", path: "notes/../.numen/config.json", ours: true},
		{name: "out of the service folder the long way", path: ".numen/../notes/a.md", vault: true},
		{name: "out of the vault", path: "notes/../../etc/passwd"},
		{name: "absolute", path: "/etc/passwd"},
		{name: "empty", path: ""},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			if _, _, err := within(real, c.path, DefaultServiceDir); (err == nil) != c.vault {
				t.Errorf("within(%q) gave %v, want accepted=%v", c.path, err, c.vault)
			}
			if _, _, err := service(real, c.path, DefaultServiceDir); (err == nil) != c.ours {
				t.Errorf("service(%q) gave %v, want accepted=%v", c.path, err, c.ours)
			}
		})
	}
}
