package markdown

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// A line opening with `#` inside a block scalar is what the person wrote there.
// Changing the entry it stands in writes it once, in the value it belongs to.
func TestAHashLineInsideABlockScalarIsTheValue(t *testing.T) {
	for name, raw := range map[string]string{
		"entry last in the block": "---\n" +
			"links:\n" +
			"  - to: A\n" +
			"    role: ref\n" +
			"    note: |\n" +
			"      the first thought\n" +
			"      # the second one\n" +
			"---\n" +
			"body\n",
		"an entry below it": "---\n" +
			"links:\n" +
			"  - to: A\n" +
			"    role: ref\n" +
			"    note: |\n" +
			"      the first thought\n" +
			"      # the second one\n" +
			"  - to: B\n" +
			"    role: ref\n" +
			"---\n" +
			"body\n",
	} {
		t.Run(name, func(t *testing.T) {
			d, err := Open([]byte(raw))
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			if _, err := d.UpdateLink(domain.ParseAddress("A"), domain.Link{
				Role: domain.RoleParent,
			}); err != nil {
				t.Fatalf("update: %v", err)
			}

			got := string(d.Bytes())
			if n := strings.Count(got, "the second one"); n != 1 {
				t.Fatalf("the line stands %d times, want once\n%s", n, got)
			}
			again, err := Open([]byte(got))
			if err != nil {
				t.Fatalf("the note stopped being readable: %v\n%s", err, got)
			}
			links, err := again.Links()
			if err != nil {
				t.Fatalf("links: %v", err)
			}
			if len(links) == 0 || links[0].Note != "the first thought\n# the second one" {
				t.Errorf("note = %q\n%s", links[0].Note, got)
			}
		})
	}
}
