package filesystem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// vault is a root the rules can be asked about: a folder of notes, the
// application's own folder, a link that stays inside, and a link that leaves.
// The links are what makes the question more than a question about text — a
// path that reads as the vault's still lands wherever the filesystem takes it.
func vault(tb testing.TB) string {
	tb.Helper()
	root := tb.TempDir()
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		tb.Fatal(err)
	}
	away := filepath.Join(filepath.Dir(real), "away")
	for _, dir := range []string{
		filepath.Join(real, "notes"),
		filepath.Join(real, DefaultServiceDir, "ocr"),
		away,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			tb.Fatal(err)
		}
	}
	for _, link := range [][2]string{
		{filepath.Join(real, "notes"), filepath.Join(real, "inward")},
		{away, filepath.Join(real, "outward")},
		{away, filepath.Join(real, "notes", "outward")},
	} {
		if err := os.Symlink(link[0], link[1]); err != nil {
			tb.Fatal(err)
		}
	}
	return real
}

// Everything that reaches a vault from outside the application comes through
// the two rules, so what they answer is the whole of the containment: a path is
// the vault's or the application's and never both, and whichever of them
// answers, what it hands back is under the root once every link on the way to
// it is resolved.
//
// A path arrives from a person's editor, from another machine's sync, and from
// a browser's address bar, so the bytes are a stranger's.
func FuzzInside(f *testing.F) {
	root := vault(f)
	for _, path := range paths {
		f.Add(path)
	}
	for _, path := range []string{
		"inward/Entropy.md",
		"outward/Entropy.md",
		"notes/outward/../../../etc/passwd",
		"notes/./../.numen/ocr/a.txt",
		".numen\\config.json",
		strings.Repeat("a/", 200) + "note.md",
		"\xff\xfe.md",
	} {
		f.Add(path)
	}

	f.Fuzz(func(t *testing.T, path string) {
		asVault, vaultReal, vaultErr := within(root, path, DefaultServiceDir)
		asOurs, oursReal, oursErr := service(root, path, DefaultServiceDir)

		if vaultErr == nil && oursErr == nil {
			t.Fatalf("%q is both the vault's (%s) and the application's (%s)",
				path, asVault, asOurs)
		}
		held(t, root, path, "within", asVault, vaultReal, vaultErr)
		held(t, root, path, "service", asOurs, oursReal, oursErr)

		// A path that could not name anything inside a vault is refused before
		// the filesystem is asked anything at all.
		if strings.ContainsRune(path, 0) || !filepath.IsLocal(path) {
			if vaultErr == nil || oursErr == nil {
				t.Fatalf("%q names nothing a vault holds and was answered for", path)
			}
		}

		// The two pairs the package reads a vault through say the same thing as
		// the rule underneath them.
		if got, err := inside(root, path, DefaultServiceDir); err != nil != (vaultErr != nil) || got != asVault {
			t.Fatalf("inside(%q) gave %q, %v and within gave %q, %v", path, got, err, asVault, vaultErr)
		}
		if got, err := followed(root, path, DefaultServiceDir); err != nil != (vaultErr != nil) || got != vaultReal {
			t.Fatalf("followed(%q) gave %q, %v and within gave %q, %v", path, got, err, vaultReal, vaultErr)
		}
	})
}

// held fails unless a rule that answered handed back a path under the root,
// both where it lands and where it lands with every link resolved, and unless a
// rule that refused handed back nothing at all.
func held(t *testing.T, root, path, rule, target, real string, err error) {
	t.Helper()
	if err != nil {
		if target != "" || real != "" {
			t.Fatalf("%s refused %q and answered with %q, %q", rule, path, target, real)
		}
		return
	}
	for what, got := range map[string]string{"lands at": target, "resolves to": real} {
		if !under(got, root) {
			t.Fatalf("%s took %q, which %s %q, outside %q", rule, path, what, got, root)
		}
	}
}
