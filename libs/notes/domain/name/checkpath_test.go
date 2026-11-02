package name

import (
	"errors"
	"testing"
)

func TestCheckPathRefusesNamesNoFileCanHave(t *testing.T) {
	for _, path := range []string{
		"note:dd.md", "a/note:dd.md", `a\b.md`, "star*.md", "what?.md", `quote".md`,
		"<in>.md", "a|b.md", "dd.md ", "dd.", "folder /dd.md", "folder./dd.md",
	} {
		if err := CheckPath(path); !errors.Is(err, ErrInvalidName) {
			t.Errorf("CheckPath(%q) = %v, want ErrInvalidName", path, err)
		}
	}
}

func TestCheckPathKeepsOrdinaryPaths(t *testing.T) {
	for _, path := range []string{
		"dd.md", "Books/x.md", "notes/Мысли о приливе.md", "two words.md", ".hidden/a.md", "a/../b.md", "a/#tag.md", "",
	} {
		if err := CheckPath(path); err != nil {
			t.Errorf("CheckPath(%q) = %v, want nil", path, err)
		}
	}
}
