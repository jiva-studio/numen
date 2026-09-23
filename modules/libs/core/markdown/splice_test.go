package markdown

import "testing"

// YAML ends a document at `...`, so what stands below that line is inside the
// delimiters and belongs to no key. It is the person's, and a write to the key
// above it leaves it where it is.
func TestBytesBelongingToNoKeySurviveAWrite(t *testing.T) {
	raw := "---\n" +
		"title: Old\n" +
		"...\n" +
		"my own scratch notes, not yaml: [[[\n" +
		"keep me\n" +
		"---\n" +
		"body\n"

	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := d.SetTitle("New"); err != nil {
		t.Fatalf("set title: %v", err)
	}

	want := "---\n" +
		"title: New\n" +
		"...\n" +
		"my own scratch notes, not yaml: [[[\n" +
		"keep me\n" +
		"---\n" +
		"body\n"
	if got := string(d.Bytes()); got != want {
		t.Errorf("a write took bytes that were not the key's\n want %q\n  got %q", want, got)
	}
}
