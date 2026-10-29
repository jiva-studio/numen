package markdown

import (
	"errors"
	"testing"
)

// A splice inserts bytes into a block with an existing layout. When the result
// is not valid YAML, the write fails and the note is unchanged.
func TestAWritePreservingUnreadableFrontmatterIsRefused(t *testing.T) {
	for name, raw := range map[string]string{
		"a document end below the key": "---\n0: \n... 0\n---\nbody\n",
		"a key written in":             "---\n 0:\n0\n---\nbody\n",
	} {
		t.Run(name, func(t *testing.T) {
			d, err := Open([]byte(raw))
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			if err := d.SetTitle("New"); !errors.Is(err, ErrUnreadable) {
				t.Errorf("want ErrUnreadable, got %v", err)
			}
			if got := string(d.Bytes()); got != raw {
				t.Errorf("a failed write changed the note\n want %q\n  got %q", raw, got)
			}
		})
	}
}
