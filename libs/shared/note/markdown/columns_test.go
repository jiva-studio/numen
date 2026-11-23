package markdown

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

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
	start, end, ok := d.findScalarSpan(value)
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
			if lastIdx := strings.LastIndex(one.front, one.want); start != lastIdx {
				t.Errorf("the span of %s in %q opens at %d, want %d", one.key, one.front, start, lastIdx)
			}
			if got != one.want {
				t.Errorf("the span of %s in %q is %q, want %q", one.key, one.front, got, one.want)
			}
		})
	}
}

// A link is retargeted in a note written in another script, and every byte
// outside the entry is unchanged.
func TestALinkMovesInANoteSavedInAnotherScript(t *testing.T) {
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
			moved, err := d.RedirectLinks(domain.ParseAddress(one.from), one.to)
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

func TestEndingsAndFrontmatterSpanBounds(t *testing.T) {
	if !hasTrailingBreak([]byte("\n")) || !hasTrailingBreak([]byte("\r\n")) || hasTrailingBreak([]byte("x")) {
		t.Errorf("endsWithBreak unexpected result")
	}

	doc, err := Open([]byte("---\n---\n"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := doc.SetMapping("meta", []Entry{{Key: "author", Value: "Ada"}}); err != nil {
		t.Fatalf("SetMapping: %v", err)
	}
	if got, want := string(doc.Bytes()), "---\nmeta:\n  author: Ada\n---\n"; got != want {
		t.Errorf("SetMapping indent: got %q, want %q", got, want)
	}

	d := &Document{front: []byte("a: |\n  indented\nb: 2\n")}
	if _, _, found := d.findSpan(&yaml.Node{
		Kind:    yaml.MappingNode,
		Content: []*yaml.Node{{Value: "orphan", Line: 1, Column: 1}},
	}, "orphan"); found {
		t.Errorf("expected odd-length Content node not to match")
	}

	start, end, found := d.findSpan(&yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Value: "a", Line: 1, Column: 1},
			{Kind: yaml.ScalarNode, Line: 1},
			{Value: "b", Line: 2, Column: 1},
			{Kind: yaml.ScalarNode, Line: 3},
		},
	}, "a")
	if !found || start != 0 || end != 5 {
		t.Errorf("clamped span to next key: got %d..%d (%v), want 0..5", start, end, found)
	}

	dShort := &Document{front: []byte("a: 1\n")}
	start, end, found = dShort.findSpan(&yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Value: "a", Line: 1, Column: 1},
			{Kind: yaml.MappingNode, Line: 10},
		},
	}, "a")
	if !found || start != 0 || end != 5 {
		t.Errorf("clamped span to lines bound: got %d..%d (%v), want 0..5", start, end, found)
	}

	lines := getLineOffsets(d.front)
	if got := d.endLine(lines, &yaml.Node{
		Kind:    yaml.MappingNode,
		Line:    0,
		Content: []*yaml.Node{{Line: 1, Column: 1}, {Kind: yaml.MappingNode, Line: 2}},
	}, 1); got != 0 {
		t.Errorf("endLine for Line 0 = %d, want 0", got)
	}

	if got := d.endLineOfMapping(lines, &yaml.Node{
		Kind: yaml.MappingNode,
		Line: 1,
		Content: []*yaml.Node{
			{Line: 1, Column: 1},
			{Kind: yaml.MappingNode, Line: 1},
			{Line: 10, Column: 1},
			{Kind: yaml.MappingNode, Line: 3},
		},
	}); got != 3 {
		t.Errorf("endLineOfMapping = %d, want 3", got)
	}

	_, err = Open([]byte("---\n: [\n---\n"))
	if multi, ok := errors.AsType[interface {
		error
		Unwrap() []error
	}](err); !ok || len(multi.Unwrap()) != 2 {
		t.Errorf("expected dual-wrapped error from Open, got %v", err)
	}
}

func TestKeysAndLinkEntryBounds(t *testing.T) {
	d := &Document{
		mapping: &yaml.Node{
			Kind:    yaml.MappingNode,
			Content: []*yaml.Node{{Value: "title"}},
		},
	}
	if got, ok := d.GetTitle(); ok || got != "" {
		t.Errorf("Title on odd Content = %q, %v", got, ok)
	}

	dOdd := &Document{
		mapping: &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Value: "meta"},
				{Kind: yaml.MappingNode, Content: []*yaml.Node{{Value: "orphan"}}},
			},
		},
	}
	if names, ok := dOdd.ListEntryNames("meta"); !ok || len(names) != 0 {
		t.Errorf("EntryNames on odd Content = %v, %v", names, ok)
	}

	doc, err := Open([]byte("---\nmeta:\n  a: 1\n  b: 2\n  c: 3\n---\n"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if names, ok := doc.ListEntryNames("meta"); !ok || !slices.Equal(names, []string{"a", "b", "c"}) {
		t.Errorf("EntryNames = %v, %v, want [a b c]", names, ok)
	}

	docQuotedMap, err := Open([]byte("---\ntags:\n  \"a\": 1\n---\n"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := docQuotedMap.SetList("tags", []string{"a"}); err != nil {
		t.Fatalf("SetList: %v", err)
	}
	if got, want := string(docQuotedMap.Bytes()), "---\ntags:\n  - a\n---\n"; got != want {
		t.Errorf("SetList over mapping: got %q, want %q", got, want)
	}

	var item yaml.Node
	if err := yaml.Unmarshal([]byte("to: Target\ntype: to\nrole: ref\n"), &item); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got := linkOf(item.Content[0]); got.Target.Value != "Target" || got.Type != "to" || got.Role != domain.RoleRef {
		t.Errorf("linkOf = %+v", got)
	}

	if !hasOnlyOwnedKeys(&yaml.Node{
		Kind:    yaml.MappingNode,
		Content: []*yaml.Node{{Value: "unowned"}},
	}) {
		t.Errorf("expected odd-length Content node to be ignored by hasOnlyOwnedKeys")
	}

	if got := keepLinkNotation("[[Old#Section]]", "New"); got != "[[New#Section]]" {
		t.Errorf("keepLinkNotation = %q, want [[New#Section]]", got)
	}
}

func TestLinkBlockBoundsAndErrors(t *testing.T) {
	d := &Document{front: []byte("links:\n  - to: A\n  - to: B\n  - to: C\n")}
	entries, _ := d.readBlockEntries(&yaml.Node{
		Kind:   yaml.SequenceNode,
		Column: 3,
		Content: []*yaml.Node{
			{Kind: yaml.MappingNode, Line: 5},
			{
				Kind: yaml.MappingNode,
				Line: 2,
				Content: []*yaml.Node{
					{Value: "to", Line: 2, Column: 5},
					{Kind: yaml.MappingNode, Line: 4, Value: "A"},
				},
			},
			{Kind: yaml.MappingNode, Line: 3},
			{
				Kind: yaml.MappingNode,
				Line: 2,
				Content: []*yaml.Node{
					{Value: "to", Line: 2, Column: 5},
					{Kind: yaml.MappingNode, Line: 4, Value: "C"},
				},
			},
		},
	}, 27)
	if len(entries) != 3 || entries[0].end != 17 || entries[2].end != 27 {
		t.Errorf("readBlockEntries = %+v", entries)
	}

	docAdd, err := Open([]byte("---\nlinks:\n  - to: A\n    role: ref\n    type: other\n  - to: A\n    role: ref\n    type: match\n---\n"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := docAdd.AddLink(domain.Link{Target: domain.ParseAddress("A"), Type: "match", Role: domain.RoleJump}); err != nil {
		t.Fatalf("AddLink: %v", err)
	}
	wantAdd := "---\nlinks:\n  - to: A\n    role: ref\n    type: other\n  - to: A\n    role: jump\n    type: match\n---\n"
	if got := string(docAdd.Bytes()); got != wantAdd {
		t.Errorf("AddLink got %q, want %q", got, wantAdd)
	}

	unterm, err := Open([]byte("---\ntitle: A\n"))
	if err != nil {
		t.Fatalf("Open unterminated: %v", err)
	}
	if err := unterm.SetLink(domain.Link{Target: domain.ParseAddress("A"), Role: domain.RoleRef}); !errors.Is(err, ErrUnterminated) {
		t.Errorf("SetLink on unterminated = %v", err)
	}
	if n, err := unterm.RemoveLink(domain.LinkKey{Target: domain.ParseAddress("A")}); !errors.Is(err, ErrUnterminated) || n != 0 {
		t.Errorf("RemoveLink on unterminated = %d, %v", n, err)
	}
	if err := unterm.SetLinkOfType("preset", domain.ParseAddress("A"), domain.RoleRef); !errors.Is(err, ErrUnterminated) {
		t.Errorf("SetLinkOfType on unterminated = %v", err)
	}
	if n, err := unterm.UpdateLink(domain.ParseAddress("A"), domain.Link{Role: domain.RoleJump}); !errors.Is(err, ErrUnterminated) || n != 0 {
		t.Errorf("UpdateLink on unterminated = %d, %v", n, err)
	}
	if n, err := unterm.RedirectLinks(domain.ParseAddress("A"), "B"); !errors.Is(err, ErrUnterminated) || n != 0 {
		t.Errorf("RedirectLinks on unterminated = %d, %v", n, err)
	}

	if carrying, retained, err := findTypedEntries([]entry{{
		link:       domain.Link{Target: domain.ParseAddress("A"), Type: "preset"},
		isReadable: true,
		isOwned:    false,
	}}, "preset"); !errors.Is(err, ErrNotOwned) || carrying != nil || retained != 0 {
		t.Errorf("findTypedEntries unowned = %v, %d, %v", carrying, retained, err)
	}

	docClearType, err := Open([]byte("---\nlinks:\n  - to: A\n    role: ref\n    type: preset\n  - to: B\n    role: ref\n    type: other\n---\n"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := docClearType.SetLinkOfType("preset", domain.Address{}, ""); err != nil {
		t.Fatalf("SetLinkOfType clear: %v", err)
	}
	wantClearType := "---\nlinks:\n  - to: B\n    role: ref\n    type: other\n---\n"
	if got := string(docClearType.Bytes()); got != wantClearType {
		t.Errorf("SetLinkOfType clear with retained entry: got %q, want %q", got, wantClearType)
	}

	docRedirect, err := Open([]byte("---\nlinks:\n  - to: Ferns\n    role: ref\n  - to: |-\n      Ferns\n    role: ref\n---\n"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	moved, err := docRedirect.RedirectLinks(domain.ParseAddress("Ferns"), "Mosses")
	if err != nil || moved != 1 {
		t.Fatalf("RedirectLinks: moved=%d, err=%v", moved, err)
	}
	if !strings.Contains(string(docRedirect.Bytes()), "- to: Mosses\n") {
		t.Errorf("expected first entry to be redirected: %q", string(docRedirect.Bytes()))
	}
}

func TestParseRangesScalarAndWikilinksBounds(t *testing.T) {
	if got := Parse(domain.FileStat{}, []byte("not-delim\n---\n")).Body; got != "not-delim\n---\n" {
		t.Errorf("Body without opening delimiter = %q", got)
	}

	for _, text := range []string{"", "plain", "####### seven", "#no-space", "#   "} {
		if level, heading, ok := parseHeading(text); ok || level != 0 || heading != "" {
			t.Errorf("parseHeading(%q) = %d, %q, %v", text, level, heading, ok)
		}
	}

	if r, rep := Diff(" a", "b"); r != (Range{Start: 0, End: 2}) || rep != "b" {
		t.Errorf("Diff(\" a\", \"b\") = %+v, %q", r, rep)
	}
	if canSpanBeginAt(" \x80", 1) {
		t.Errorf("canSpanBeginAt should reject continuation byte")
	}
	if canSpanEndAt("\x80 ", 0) {
		t.Errorf("canSpanEndAt should reject continuation byte")
	}

	d := &Document{front: []byte("a: 1\n")}
	if start, end, ok := d.findScalarSpan(&yaml.Node{Kind: yaml.MappingNode, Line: 1, Column: 1, Value: "a"}); ok || start != 0 || end != 0 {
		t.Errorf("scalarSpan on MappingNode = %d, %d, %v", start, end, ok)
	}
	if end, ok := findQuotedEnd([]byte("a'"), 0, '\''); ok || end != 0 {
		t.Errorf("findQuotedEnd without opening quote = %d, %v", end, ok)
	}
	if end, ok := findQuotedEnd([]byte("'unclosed"), 0, '\''); ok || end != 0 {
		t.Errorf("findQuotedEnd unclosed = %d, %v", end, ok)
	}

	foldedOut, err := encodeScalarLike("a\nb", yaml.FoldedStyle)
	if err != nil {
		t.Fatalf("scalarLike: %v", err)
	}
	var roundtrip string
	if err := yaml.Unmarshal([]byte(foldedOut), &roundtrip); err != nil || roundtrip != "a\nb" {
		t.Errorf("scalarLike folded roundtrip = %q (%v), want \"a\\nb\"", roundtrip, err)
	}

	links := FindWikilinks("[[]][[A]]")
	if len(links) != 1 || links[0].Inside != "A" {
		t.Errorf("WikilinksIn(\"[[]][[A]]\") = %+v", links)
	}
}
