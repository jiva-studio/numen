package note_test

import (
	"testing"

	"golang.org/x/text/unicode/norm"
)

// One name each, composed and decomposed. The two forms are computed rather
// than typed out: these become filenames, and an editor that composed this file
// on the way in would leave the tests testing nothing.
var (
	hangulComposed   = norm.NFC.String("한글")
	hangulDecomposed = norm.NFD.String("한글")
	kanaComposed     = norm.NFC.String("がき")
	kanaDecomposed   = norm.NFD.String("がき")
)

// A link written in one case reaches the note filed under another, in every
// script that has cases.
func TestANameReachesItsNoteWhateverCaseItIsWrittenIn(t *testing.T) {
	for _, one := range []struct {
		what    string
		written string
		filed   string
	}{
		{"Cyrillic", "энтропия", "Энтропия.md"},
		{"an acute", "CAFÉ", "Café.md"},
		{"a Greek final sigma", "ὈΔΥΣΣΕΎΣ", "ὀδυσσεύς.md"},
		{"a German sharp s", "STRASSE", "straße.md"},
	} {
		t.Run(one.what, func(t *testing.T) {
			db, v := newIndexedVault(t, map[string]string{
				"source.md": "Points at [[" + one.written + "]].\n",
				one.filed:   "# The note\n",
			})

			c := links(t, db, v, "source.md")
			if len(c.Links) != 1 {
				t.Fatalf("got %+v", c.Links)
			}
			if c.Links[0].To != one.filed {
				t.Errorf("[[%s]] resolved to %q, want %q", one.written, c.Links[0].To, one.filed)
			}
		})
	}
}

// A script with no case is carried by the normal form alone: the two spellings
// of one Korean or Japanese name are different bytes and one name.
func TestANameReachesItsNoteWhicheverWayItIsComposed(t *testing.T) {
	for _, one := range []struct {
		what    string
		written string
		filed   string
	}{
		{"a composed link to a decomposed name", hangulComposed, hangulDecomposed + ".md"},
		{"a decomposed link to a composed name", hangulDecomposed, hangulComposed + ".md"},
		{"a voiced mark written as one character", kanaComposed, kanaDecomposed + ".md"},
		{"a voiced mark written as two", kanaDecomposed, kanaComposed + ".md"},
	} {
		t.Run(one.what, func(t *testing.T) {
			if one.written+".md" == one.filed {
				t.Fatal("the link and the filename are the same bytes, so nothing is tested")
			}
			db, v := newIndexedVault(t, map[string]string{
				"source.md": "Points at [[" + one.written + "]].\n",
				one.filed:   "# The note\n",
			})

			c := links(t, db, v, "source.md")
			if len(c.Links) != 1 {
				t.Fatalf("got %+v", c.Links)
			}
			if c.Links[0].To != one.filed {
				t.Errorf("[[%s]] resolved to %q, want %q", one.written, c.Links[0].To, one.filed)
			}
		})
	}
}

// The folders on the way to a note answer the way the note's own name does.
func TestAPathReachesItsNoteWhateverCaseItIsWrittenIn(t *testing.T) {
	db, v := newIndexedVault(t, map[string]string{
		"source.md":           "Points at [[ЗАМЕТКИ/ЭНТРОПИЯ]].\n",
		"Заметки/Энтропия.md": "# The one that was asked for\n",
		"архив/Энтропия.md":   "# The stranger\n",
	})

	c := links(t, db, v, "source.md")
	if len(c.Links) != 1 {
		t.Fatalf("got %+v", c.Links)
	}
	if c.Links[0].To != "Заметки/Энтропия.md" {
		t.Errorf("resolved to %q, want the path that was written", c.Links[0].To)
	}
	if c.Links[0].IsAmbiguous {
		t.Error("a path that was written out was called ambiguous")
	}
}

// The note a link reaches lists it back.
func TestANoteListsABacklinkWrittenInAnotherCase(t *testing.T) {
	db, v := newIndexedVault(t, map[string]string{
		"source.md":   "Points at [[энтропия]].\n",
		"Энтропия.md": "# Энтропия\n",
	})

	back := links(t, db, v, "Энтропия.md")
	if len(back.Backlinks) != 1 {
		t.Fatalf("backlinks = %+v", back.Backlinks)
	}
	if back.Backlinks[0].From != "source.md" {
		t.Errorf("the backlink comes from %q", back.Backlinks[0].From)
	}
}

// A dotted capital I is a letter of its own outside Turkish, so it names a note
// of its own.
func TestATurkishDottedIIsAName(t *testing.T) {
	db, v := newIndexedVault(t, map[string]string{
		"source.md":   "Points at [[İstanbul]].\n",
		"istanbul.md": "# Somewhere else\n",
	})

	c := links(t, db, v, "source.md")
	if len(c.Links) != 1 {
		t.Fatalf("got %+v", c.Links)
	}
	if c.Links[0].To != "" {
		t.Errorf("[[İstanbul]] resolved to %q", c.Links[0].To)
	}
}
