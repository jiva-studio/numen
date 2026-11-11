package domain

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
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
		"dd.md", "Books/x.md", "notes/Мысли о приливе.md", "two words.md", ".hidden/a.md", "./a.md", "a/../b.md", "a/#tag.md", "",
	} {
		if err := CheckPath(path); err != nil {
			t.Errorf("CheckPath(%q) = %v, want nil", path, err)
		}
	}
}

func TestFilenameAndLengthBoundary(t *testing.T) {
	for _, title := range []string{"", "   ", "...", "\x00", ". ."} {
		name, exact, err := Filename(title)
		if !errors.Is(err, ErrInvalidName) || name != "" || exact {
			t.Errorf("Filename(%q) = %q, %v, %v", title, name, exact, err)
		}
	}
	for _, title := range []string{"first\nsecond", "first\rsecond", "first\r\nsecond"} {
		name, exact, err := Filename(title)
		if !errors.Is(err, ErrInvalidName) || name != "" || exact {
			t.Errorf("Filename(%q) = %q, %v, %v", title, name, exact, err)
		}
	}
	if got, exact, err := Filename("Entropy"); err != nil || got != "Entropy" || !exact {
		t.Errorf("Filename(\"Entropy\") = %q, %v, %v", got, exact, err)
	}
	if got, exact, err := Filename("a/b"); err != nil || got != "a-b" || exact {
		t.Errorf("Filename(\"a/b\") = %q, %v, %v", got, exact, err)
	}
	if got, exact := GetReducedFilename(`a/b\c:d*e?f"g<h>i|j#k`); got != "a-b-c-d-e-f-g-h-i-j-k" || exact {
		t.Errorf("GetReducedFilename reserved = %q, %v", got, exact)
	}
	if got, exact := GetReducedFilename("Notes [[draft]]"); got != "Notes [draft]" || exact {
		t.Errorf("GetReducedFilename brackets = %q, %v", got, exact)
	}
	if got, exact := GetReducedFilename("a [1980] note"); got != "a [1980] note" || !exact {
		t.Errorf("GetReducedFilename single brackets = %q, %v", got, exact)
	}
	if got, exact := GetReducedFilename("En\x00tro\npy"); got != "Entropy" || exact {
		t.Errorf("GetReducedFilename control = %q, %v", got, exact)
	}
	for _, title := range []string{"Entropy.", "Entropy ", "Entropy. ."} {
		got, exact := GetReducedFilename(title)
		if got != "Entropy" || exact != (title == "Entropy ") {
			t.Errorf("%q -> %q, %v", title, got, exact)
		}
	}
	for title, device := range map[string]bool{
		"CON": true, "PRN": true, "AUX": true, "NUL": true,
		"COM1": true, "COM9": true, "LPT1": true, "LPT9": true,
		"con.txt": true, "COM0": false, "LPT0": false, "COM!": false,
		"COMA": false, "COM12": false, "COM": false, "XYZ1": false, "CO1": false,
	} {
		got, exact := GetReducedFilename(title)
		if (got != title) != device || exact == device {
			t.Errorf("%q -> %q, %v (want device=%v)", title, got, exact, device)
		}
	}
	longest := strings.Repeat("x", 120)
	if got, exact := GetReducedFilename(longest); got != longest || !exact {
		t.Errorf("120 bytes: got %d, exact %v", len(got), exact)
	}
	if got, exact := GetReducedFilename(longest + "x"); got != longest || exact {
		t.Errorf("121 bytes: got %d, exact %v", len(got), exact)
	}
	for _, char := range []string{"é", "ṛ", "😀"} {
		for pad := range 4 {
			title := strings.Repeat("a", pad) + strings.Repeat(char, 150)
			got, exact := GetReducedFilename(title)
			if exact || !utf8.ValidString(got) || !strings.HasPrefix(title, got) {
				t.Fatalf("%d+%q: got %d bytes, exact %v", pad, char, len(got), exact)
			}
			if len(got) > 120 || len(got)+len(char) <= 120 {
				t.Errorf("pad %d, %q: cut at %d bytes", pad, char, len(got))
			}
		}
	}
	if got := truncateAtRuneBoundary("short", 10); got != "short" {
		t.Errorf("truncateAtRuneBoundary short = %q", got)
	}
	if got := truncateAtRuneBoundary("\x80\x80", 1); got != "" {
		t.Errorf("truncateAtRuneBoundary continuation = %q", got)
	}
}
