package markdown

import (
	"reflect"
	"strings"
	"testing"
)

var fenceScanBodies = map[string]string{
	"plain":        "# One\n\ntext [[A]]\n\n## Two [[B]]\n",
	"fenced":       "# One\n```\n# Not\n[[Hidden]]\n```\n## Two [[B]]\n",
	"tilde":        "~~~\n# Not\n[[Hidden]]\n~~~\n# Yes [[C]]\n",
	"unterminated": "# One\n```go\n# Not\n[[Hidden]]\n## Not either\n",
	"indented":     "# One\n   ```\n# Not\n[[Hidden]]\n   ```\n## Two [[B]]\n",
	"in a list":    "- item\n  ```\n  # Not\n  [[Hidden]]\n  ```\n- item [[D]]\n\n# After\n",
	"mixed marks":  "```\n~~~\n# Not\n[[Hidden]]\n~~~\n```\n# After [[E]]\n",
	"empty":        "",
	"no newline":   "# Only [[F]]",
}

func TestFenceDetectionIsTheSameForCRLFAndLF(t *testing.T) {
	for name, lf := range fenceScanBodies {
		t.Run(name, func(t *testing.T) {
			crlf := strings.ReplaceAll(lf, "\n", "\r\n")

			lfHeadings, crlfHeadings := headings([]byte(lf)), headings([]byte(crlf))
			if len(lfHeadings) != len(crlfHeadings) {
				t.Fatalf("LF has %d headings, CRLF has %d", len(lfHeadings), len(crlfHeadings))
			}
			for i := range lfHeadings {
				if lfHeadings[i].Text != crlfHeadings[i].Text ||
					lfHeadings[i].Level != crlfHeadings[i].Level ||
					lfHeadings[i].Line != crlfHeadings[i].Line {
					t.Errorf("heading %d differs: %+v and %+v", i, lfHeadings[i], crlfHeadings[i])
				}
			}
			if got, want := bodyLinks([]byte(crlf)), bodyLinks([]byte(lf)); !reflect.DeepEqual(got, want) {
				t.Errorf("CRLF links %v, LF links %v", got, want)
			}
		})
	}
}

func TestHeadingsAndLinksSkipWhatAFenceHolds(t *testing.T) {
	cases := []struct {
		body     string
		headings []string
		links    []string
	}{
		{fenceScanBodies["fenced"], []string{"One", "Two [[B]]"}, []string{"name://B"}},
		{fenceScanBodies["tilde"], []string{"Yes [[C]]"}, []string{"name://C"}},
		{fenceScanBodies["unterminated"], []string{"One"}, nil},
		{fenceScanBodies["indented"], []string{"One", "Two [[B]]"}, []string{"name://B"}},
		{fenceScanBodies["in a list"], []string{"After"}, []string{"name://D"}},
		{fenceScanBodies["mixed marks"], []string{"After [[E]]"}, []string{"name://E"}},
		{fenceScanBodies["no newline"], []string{"Only [[F]]"}, []string{"name://F"}},
	}
	for _, c := range cases {
		var gotHeadings, gotLinks []string
		for _, h := range headings([]byte(c.body)) {
			gotHeadings = append(gotHeadings, h.Text)
		}
		for _, l := range bodyLinks([]byte(c.body)) {
			gotLinks = append(gotLinks, l.Target.String())
		}
		if !reflect.DeepEqual(gotHeadings, c.headings) {
			t.Errorf("%q: headings %v, want %v", c.body, gotHeadings, c.headings)
		}
		if !reflect.DeepEqual(gotLinks, c.links) {
			t.Errorf("%q: links %v, want %v", c.body, gotLinks, c.links)
		}
	}
}
