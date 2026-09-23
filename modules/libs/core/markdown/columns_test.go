package markdown

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// spanOf is where one top-level key's value stands in the frontmatter, and the
// bytes it occupies.
func spanOf(t *testing.T, front, key string) (int, string, bool) {
	t.Helper()
	d, err := Open([]byte("---\n" + front + "---\nbody\n"))
	if err != nil {
		t.Fatalf("opening %q: %v", front, err)
	}
	node, err := d.readMapping()
	if err != nil || node == nil {
		t.Fatalf("reading %q: %v", front, err)
	}
	value := valueOf(node, key)
	if value == nil {
		t.Fatalf("%q holds no %s", front, key)
	}
	start, end, ok := d.scalarSpan(value)
	if !ok {
		return 0, "", false
	}
	return start, string(d.front[start:end]), true
}

// The parser counts a column in characters, and the span it addresses is bytes.
// A value standing after Cyrillic, Greek or Chinese on its line is the same
// token as one standing after Latin.
func TestAScalarSpanIsTheValueAndNothingBeforeIt(t *testing.T) {
	for name, one := range map[string]struct{ front, key, want string }{
		"latin":                {"road: Ferns\n", "road", "Ferns"},
		"cyrillic key":         {"тропа: Ferns\n", "тропа", "Ferns"},
		"greek key":            {"μονοπάτι: Ferns\n", "μονοπάτι", "Ferns"},
		"chinese key":          {"小径: Ferns\n", "小径", "Ferns"},
		"cyrillic value":       {"тропа: Папоротники\n", "тропа", "Папоротники"},
		"quoted after latin":   {"road: \"Ferns\"\n", "road", "\"Ferns\""},
		"quoted after greek":   {"μονοπάτι: \"Ferns\"\n", "μονοπάτι", "\"Ferns\""},
		"quoted after chinese": {"小径: 'Ferns'\n", "小径", "'Ferns'"},
		"quoted mixed script":  {"тропа μονοπάτι 小径: \"Папоротники\"\n", "тропа μονοπάτι 小径", "\"Папоротники\""},
		// The key holds the letter the value is, so a span landing short of the
		// value reads back as the value.
		"a letter of the key repeats the value": {"ё中a: a\n", "ё中a", "a"},
	} {
		t.Run(name, func(t *testing.T) {
			start, got, ok := spanOf(t, one.front, one.key)
			if !ok {
				t.Fatalf("%q has no span for %s", one.front, one.key)
			}
			if at := strings.LastIndex(one.front, one.want); start != at {
				t.Errorf("the span of %s in %q opens at %d, want %d", one.key, one.front, start, at)
			}
			if got != one.want {
				t.Errorf("the span of %s in %q is %q, want %q", one.key, one.front, got, one.want)
			}
		})
	}
}

// A link is pointed somewhere else in a note written in another script, and
// every byte the entry does not own stands where it stood.
func TestALinkMovesInANoteWrittenInAnotherScript(t *testing.T) {
	for name, one := range map[string]struct{ raw, from, to, want string }{
		"cyrillic": {
			raw:  "---\nзаголовок: Мхи\nlinks:\n  - to: Папоротники\n    role: ref\n---\nтекст\n",
			from: "Папоротники", to: "Хвощи",
			want: "---\nзаголовок: Мхи\nlinks:\n  - to: Хвощи\n    role: ref\n---\nтекст\n",
		},
		"greek quoted": {
			raw:  "---\nτίτλος: Βρύα\nlinks:\n  - to: \"[[Φτέρες]]\"\n    role: ref\n---\nκείμενο\n",
			from: "[[Φτέρες]]", to: "Ισοέτες",
			want: "---\nτίτλος: Βρύα\nlinks:\n  - to: \"[[Ισοέτες]]\"\n    role: ref\n---\nκείμενο\n",
		},
		"chinese": {
			raw:  "---\n标题: 苔藓\nlinks:\n  - to: 蕨类\n    role: ref\n---\n正文\n",
			from: "蕨类", to: "木贼",
			want: "---\n标题: 苔藓\nlinks:\n  - to: 木贼\n    role: ref\n---\n正文\n",
		},
		"latin": {
			raw:  "---\ntitle: Mosses\nlinks:\n  - to: Ferns\n    role: ref\n---\ntext\n",
			from: "Ferns", to: "Horsetails",
			want: "---\ntitle: Mosses\nlinks:\n  - to: Horsetails\n    role: ref\n---\ntext\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			d, err := Open([]byte(one.raw))
			if err != nil {
				t.Fatalf("opening: %v", err)
			}
			moved, err := d.PointLinksAt(domain.ParseAddress(one.from), one.to)
			if err != nil {
				t.Fatalf("pointing: %v", err)
			}
			if moved != 1 {
				t.Fatalf("moved %d links, want 1", moved)
			}
			if got := string(d.Bytes()); got != one.want {
				t.Errorf("got %q, want %q", got, one.want)
			}
		})
	}
}

// A key's span is the lines its own value continues onto. A note whose keys are
// written in another script keeps every byte outside the one being written.
func TestAKeyInAnotherScriptOwnsItsOwnLines(t *testing.T) {
	raw := "---\nзаметка: |\n  первая мысль\n  вторая мысль\ntitle: Мхи\nμονοπάτι: Φτέρες\n---\nтекст\n"
	want := "---\nзаметка: |\n  первая мысль\n  вторая мысль\ntitle: Папоротники\nμονοπάτι: Φτέρες\n---\nтекст\n"

	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("opening: %v", err)
	}
	if err := d.SetTitle("Папоротники"); err != nil {
		t.Fatalf("writing the title: %v", err)
	}
	if got := string(d.Bytes()); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
