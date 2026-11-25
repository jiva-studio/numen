package note_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
)

func TestCheckRemotePath_AcceptsOrdinaryPaths(t *testing.T) {
	t.Parallel()
	for _, path := range []string{"Journal/2026-11-23.md", "Рецепты/борщ.md", "a b/c.d.md", "Конспект/console.md", "com10.md", "attachments/photo.png"} {
		if err := note.CheckRemotePath(path); err != nil {
			t.Errorf("CheckRemotePath(%q) = %v, want nil", path, err)
		}
	}
}

func TestCheckRemotePath_RefusesPathsThatLeaveOrBreakAVault(t *testing.T) {
	t.Parallel()
	for name, path := range map[string]string{
		"empty":                      "",
		"absolute":                   "/etc/passwd",
		"parent":                     "../x.md",
		"parent in the middle":       "a/../b.md",
		"current folder":             "./x.md",
		"double slash":               "a//b.md",
		"trailing slash":             "a/b/",
		"backslash":                  `a\b.md`,
		"alternate stream":           "a.md:stream",
		"drive letter":               "C:/x.md",
		"control character":          "a\x00b.md",
		"newline":                    "a\nb.md",
		"wildcard":                   "a*.md",
		"reserved CON":               "CON",
		"reserved with extension":    "notes/nul.md",
		"reserved lower case":        "aux.txt",
		"reserved COM1":              "com1.md",
		"reserved LPT9":              "LPT9.md",
		"reserved in a folder":       "prn/x.md",
		"trailing dot":               "notes/a.md.",
		"trailing space":             "notes/a.md ",
		"folder with a trailing dot": "notes./a.md",
		"service folder":             ".numen/state.json",
		"git folder":                 "x/.git/config",
		"upper case git folder":      ".GIT/hooks/post-checkout",
		"mixed case service folder":  ".Numen/config.json",
		"nested mixed case git":      "notes/.Git/x",
		"upper case trash folder":    ".TRASH/a.md",
		"temporary file":             "a.tmp",
	} {
		if err := note.CheckRemotePath(path); !errors.Is(err, note.ErrInvalidPath) {
			t.Errorf("%s: CheckRemotePath(%q) = %v, want ErrInvalidPath", name, path, err)
		}
	}
}
