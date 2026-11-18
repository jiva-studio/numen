package markdown_test

import "testing"

func TestRemovingAKeyDropsItsLinesAndKeepsEveryOtherLineAsItWas(t *testing.T) {
	doc := openDocument(t, block)

	if err := doc.Remove("fields"); err != nil {
		t.Fatalf("remove: %v", err)
	}

	want := "---\n" +
		"id: 01J8F3K2M9QRSTVWXYZ012\n" +
		"# a note to myself\n" +
		"mine: keep me verbatim\n" +
		"zebra: 1\n" +
		"---\n" +
		"\n" +
		"## Recognize\n"
	if got := string(doc.Bytes()); got != want {
		t.Errorf("wrote\n%q\nwant\n%q", got, want)
	}
}

func TestRemovingAScalarKeyDropsItsLine(t *testing.T) {
	doc := openDocument(t, block)

	if err := doc.Remove("mine"); err != nil {
		t.Fatalf("remove: %v", err)
	}

	if got := string(doc.Bytes()); got == block || containsLine(got, "mine: keep me verbatim") {
		t.Errorf("the key was not removed:\n%q", got)
	}
}

func TestRemovingAKeyTheNoteDoesNotHaveChangesNothing(t *testing.T) {
	doc := openDocument(t, block)

	if err := doc.Remove("absent"); err != nil {
		t.Fatalf("remove: %v", err)
	}

	if got := string(doc.Bytes()); got != block {
		t.Errorf("wrote\n%q\nwant the note as it was", got)
	}
}

func TestRemovingAKeyFromANoteWithoutFrontmatterChangesNothing(t *testing.T) {
	doc := openDocument(t, "# Plain\n")

	if err := doc.Remove("tags"); err != nil {
		t.Fatalf("remove: %v", err)
	}

	if got := string(doc.Bytes()); got != "# Plain\n" {
		t.Errorf("wrote %q, want the note as it was", got)
	}
}

func containsLine(text, line string) bool {
	for start := 0; start < len(text); {
		end := start
		for end < len(text) && text[end] != '\n' {
			end++
		}
		if text[start:end] == line {
			return true
		}
		start = end + 1
	}
	return false
}
