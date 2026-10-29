package vaultconfig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/ulid"
)

// vault builds a root the rules can be queried about: a folder of notes, the
// application's own folder, a link that stays inside, a link that leaves, and a
// link into the application's own folder. The links make the check more than a
// text check: a path that reads as the vault's can still resolve elsewhere in
// the filesystem.
func vault(tb testing.TB) string {
	tb.Helper()
	root := tb.TempDir()
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		tb.Fatal(err)
	}
	away := filepath.Join(filepath.Dir(resolved), "away")
	for _, dir := range []string{
		filepath.Join(resolved, "notes"),
		filepath.Join(resolved, DefaultServiceDir, "ocr"),
		away,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			tb.Fatal(err)
		}
	}
	for _, link := range [][2]string{
		{filepath.Join(resolved, "notes"), filepath.Join(resolved, "inward")},
		{away, filepath.Join(resolved, "outward")},
		{away, filepath.Join(resolved, "notes", "outward")},
		{filepath.Join(resolved, DefaultServiceDir), filepath.Join(resolved, "member")},
	} {
		if err := os.Symlink(link[0], link[1]); err != nil {
			tb.Fatal(err)
		}
	}
	return resolved
}

// Every external access to a vault goes through the two rules, so their results
// are the whole of the containment: a path belongs to the vault or the
// application and never both, and whichever rule accepts it, the result is under
// the root once every link on the way is resolved.
//
// Paths arrive from the user's editor, another machine's sync and a browser's
// address bar, so the bytes are untrusted.
func FuzzInside(f *testing.F) {
	root := vault(f)
	for _, path := range paths {
		f.Add(path)
	}
	for _, path := range []string{
		"inward/Entropy.md",
		"outward/Entropy.md",
		"member",
		"member/ocr/abc.txt",
		"member/../notes/Entropy.md",
		"HELD/config.json",
		"notes/outward/../../../etc/passwd",
		"notes/./../.numen/ocr/a.txt",
		".numen\\config.json",
		strings.Repeat("a/", 200) + "note.md",
		"\xff\xfe.md",
	} {
		f.Add(path)
	}

	f.Fuzz(func(t *testing.T, path string) {
		asVault, vaultReal, vaultErr := resolveVaultPath(root, path, DefaultServiceDir)
		asOurs, oursReal, oursErr := resolveServicePath(root, path, DefaultServiceDir)

		if vaultErr == nil && oursErr == nil {
			t.Fatalf("%q is both the vault's (%s) and the application's (%s)",
				path, asVault, asOurs)
		}
		checkRule(t, root, path, "resolveVaultPath", false, asVault, vaultReal, vaultErr)
		checkRule(t, root, path, "service", true, asOurs, oursReal, oursErr)

		// A path that cannot name anything inside a vault fails before any
		// filesystem access.
		if strings.ContainsRune(path, 0) || !filepath.IsLocal(path) {
			if vaultErr == nil || oursErr == nil {
				t.Fatalf("%q names nothing a vault holds and was answered for", path)
			}
		}

		// The two pairs the package reads a vault through say the same thing as
		// the rule underneath them.
		if got, err := GetContainedPath(root, path, DefaultServiceDir); err != nil != (vaultErr != nil) || got != asVault {
			t.Fatalf("getContainedPath(%q) gave %q, %v and resolveVaultPath gave %q, %v",
				path, got, err, asVault, vaultErr)
		}
		if got, err := ResolveLinks(root, path, DefaultServiceDir); err != nil != (vaultErr != nil) || got != vaultReal {
			t.Fatalf("resolveLinks(%q) gave %q, %v and resolveVaultPath gave %q, %v",
				path, got, err, vaultReal, vaultErr)
		}
	})
}

// checkRule fails unless an accepting rule returned a path under the root, both
// as spelled and with every link resolved, and unless a rejecting rule returned
// nothing.
//
// Where the path resolves determines which of the two rules may accept it. A
// link is a second spelling of a location, and locations are divided between the
// two rules, so a path spelled as the vault's that resolves into the
// application's folder belongs to the application and resolveVaultPath may not
// accept it.
func checkRule(t *testing.T, root, path, rule string, application bool, target, resolved string, err error) {
	t.Helper()
	if err != nil {
		if target != "" || resolved != "" {
			t.Fatalf("%s rejected %q and answered with %q, %q", rule, path, target, resolved)
		}
		return
	}
	for what, got := range map[string]string{"lands at": target, "resolves to": resolved} {
		if !IsUnderRoot(got, root) {
			t.Fatalf("%s took %q, which %s %q, outside %q", rule, path, what, got, root)
		}
	}
	if got := isOurs(getRootRelativeName(resolved, root), DefaultServiceDir); got != application {
		where := "outside the application's folder"
		if got {
			where = "into the application's folder"
		}
		t.Fatalf("%s took %q, which resolves %s: %q", rule, path, where, resolved)
	}
}

// configSeeds are the shapes a vault's configuration arrives in: what
// Initialize writes, one that names what to leave alone, one a sync truncated
// mid-write, an identifier with a character the alphabet has not got, one a
// letter short, a name written twice, an object holding nothing, a bare null,
// the file as something other than an object, and bytes that are no
// configuration at all.
var configSeeds = []string{
	"{\n  \"v\": 1,\n  \"id\": \"01JQ8ZP4T7MXVN2K5H9RBCDEFG\"\n}\n",
	"{\"v\":1,\"id\":\"01JQ8ZP4T7MXVN2K5H9RBCDEFG\",\"ignore\":[\"attachments/\",\"*.tmp\"]}",
	"{\"v\":1,\"id\":\"01JQ8ZP4T7MXV",
	"{\"v\":1,\"id\":\"01JQ8ZP4T7MXVN2K5H9RBCDEFU\"}",
	"{\"v\":1,\"id\":\"01JQ8ZP4T7MXVN2K5H9RBCDEF\"}",
	"{\"v\":1,\"id\":\"01JQ8ZP4T7MXVN2K5H9RBCDEFG\",\"id\":\"01JQ8ZP4T7MXVN2K5H9RBCDEFH\"}",
	"{}",
	"null",
	"[1, 2, 3]",
	"\xef\xbb\xbf{\"v\":1,\"id\":\"01JQ8ZP4T7MXVN2K5H9RBCDEFG\"}",
	"\x00\xff\xfe",
	"",
}

// A folder carries an identity or it does not, and ReadConfig reports which. An
// identity it returns is one that could have been created here.
//
// Bytes that cannot be parsed are rejected as unreadable.
func FuzzReadConfig(f *testing.F) {
	for _, seed := range configSeeds {
		f.Add(seed)
	}
	// One folder is written over and over: a vault for each of a million inputs
	// is a run that never gets to the end of them.
	root := f.TempDir()
	if err := os.MkdirAll(filepath.Join(root, DefaultServiceDir), 0o755); err != nil {
		f.Fatal(err)
	}
	pos := ConfigAt(root, DefaultServiceDir)

	f.Fuzz(func(t *testing.T, raw string) {
		if err := os.WriteFile(pos, []byte(raw), 0o644); err != nil {
			t.Fatal(err)
		}

		target, err := ReadConfig(root, DefaultServiceDir)
		if err == nil && !ulid.Valid(target.ID) {
			t.Fatalf("a vault was read as carrying %q, which no vault carries", target.ID)
		}
		if errors.Is(err, ErrNotAVault) {
			t.Fatalf("%q is in the folder and it was read as never having been added", raw)
		}

		made, adding := Initialize(root, DefaultServiceDir, time.Unix(1, 0))
		switch {
		case err == nil && (adding != nil || made.ID != target.ID):
			t.Fatalf("a vault carrying %q was added again as %q, %v", target.ID, made.ID, adding)
		case err != nil && adding == nil:
			t.Fatalf("%q could not be read (%v) and the folder was added all the same", raw, err)
		}
		after, readErr := os.ReadFile(pos)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if string(after) != raw {
			t.Fatalf("%q stood in the folder and %q stands there now", raw, after)
		}
	})
}
