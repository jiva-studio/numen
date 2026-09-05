package filesystem

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/ulid"
)

// vault is a root the rules can be asked about: a folder of notes, the
// application's own folder, a link that stays inside, a link that leaves, and a
// link into the application's own folder. The links are what makes the question
// more than a question about text — a path that reads as the vault's still
// lands wherever the filesystem takes it.
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
		{filepath.Join(real, DefaultServiceDir), filepath.Join(real, "held")},
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
		"held",
		"held/ocr/abc.txt",
		"held/../notes/Entropy.md",
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
		asVault, vaultReal, vaultErr := within(root, path, DefaultServiceDir)
		asOurs, oursReal, oursErr := service(root, path, DefaultServiceDir)

		if vaultErr == nil && oursErr == nil {
			t.Fatalf("%q is both the vault's (%s) and the application's (%s)",
				path, asVault, asOurs)
		}
		held(t, root, path, "within", false, asVault, vaultReal, vaultErr)
		held(t, root, path, "service", true, asOurs, oursReal, oursErr)

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
//
// Where the path resolves to is what says which of the two rules was entitled
// to answer. A link is a second spelling for a place, and it is the place the
// two divide between them, so a path spelled as the vault's that lands in the
// application's folder is the application's and within may not take it.
func held(t *testing.T, root, path, rule string, application bool, target, real string, err error) {
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
	if got := ours(landing(real, root), DefaultServiceDir); got != application {
		where := "outside the application's folder"
		if got {
			where = "into the application's folder"
		}
		t.Fatalf("%s took %q, which resolves %s: %q", rule, path, where, real)
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

// A folder carries an identity or it does not, and ReadConfig says which. An
// identity it hands back is one that could have been made here, because every
// row of the index points at it and a row pointing at nothing belongs to no
// vault.
//
// Bytes nobody can read out are refused as unreadable and not as a folder that
// was never added, and Initialize leaves them exactly where they are: a vault
// whose file a sync truncated does not get a second identity, which would part
// it from everything already indexed under the first.
//
// The file is in a folder a person keeps in their own sync, so the bytes are a
// stranger's.
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
	at := configAt(root, DefaultServiceDir)

	f.Fuzz(func(t *testing.T, raw string) {
		if err := os.WriteFile(at, []byte(raw), 0o644); err != nil {
			t.Fatal(err)
		}

		held, err := ReadConfig(root, DefaultServiceDir)
		if err == nil && !ulid.Valid(held.ID) {
			t.Fatalf("a vault was read as carrying %q, which no vault carries", held.ID)
		}
		if errors.Is(err, ErrNotAVault) {
			t.Fatalf("%q stands in the folder and it was read as never having been added", raw)
		}

		made, adding := Initialize(root, DefaultServiceDir, time.Unix(1, 0))
		switch {
		case err == nil && (adding != nil || made.ID != held.ID):
			t.Fatalf("a vault carrying %q was added again as %q, %v", held.ID, made.ID, adding)
		case err != nil && adding == nil:
			t.Fatalf("%q could not be read (%v) and the folder was added all the same", raw, err)
		}
		after, readErr := os.ReadFile(at)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if string(after) != raw {
			t.Fatalf("%q stood in the folder and %q stands there now", raw, after)
		}
	})
}
