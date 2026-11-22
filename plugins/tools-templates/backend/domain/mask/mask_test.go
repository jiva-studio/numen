package mask_test

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-templates/backend/domain/mask"
)

type table struct {
	Parse []struct {
		Raw   string   `json:"raw"`
		Masks []string `json:"masks"`
	} `json:"parse"`
	Defaults []string `json:"defaults"`
	Match    []struct {
		Mask    string `json:"mask"`
		Path    string `json:"path"`
		Matches bool   `json:"matches"`
	} `json:"match"`
	Group []struct {
		Mask  string `json:"mask"`
		Path  string `json:"path"`
		Group string `json:"group"`
	} `json:"group"`
	Find []struct {
		Masks string  `json:"masks"`
		Path  string  `json:"path"`
		Group *string `json:"group"`
	} `json:"find"`
}

func readTable(t *testing.T) table {
	t.Helper()
	raw, err := os.ReadFile("../../../testdata/masks.json")
	if err != nil {
		t.Fatal(err)
	}
	var got table
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	return got
}

func TestParse(t *testing.T) {
	cases := readTable(t)
	for _, c := range cases.Parse {
		want := c.Masks
		if len(want) == 0 {
			want = cases.Defaults
		}
		if got := mask.Parse(c.Raw); !reflect.DeepEqual(got, mask.Parse(strings.Join(want, ","))) {
			t.Errorf("Parse(%q) = %q, want %q", c.Raw, got, want)
		}
	}
}

func TestMatch(t *testing.T) {
	for _, c := range readTable(t).Match {
		if got := mask.Parse(c.Mask)[0].Match(c.Path); got != c.Matches {
			t.Errorf("%q.Match(%q) = %v, want %v", c.Mask, c.Path, got, c.Matches)
		}
	}
}

func TestGroup(t *testing.T) {
	for _, c := range readTable(t).Group {
		if got := mask.Parse(c.Mask)[0].Group(c.Path); got != c.Group {
			t.Errorf("%q.Group(%q) = %q, want %q", c.Mask, c.Path, got, c.Group)
		}
	}
}

func TestFindUsesTheFirstMaskThatMatches(t *testing.T) {
	for _, c := range readTable(t).Find {
		group, ok := mask.Parse(c.Masks).Find(c.Path)
		if ok != (c.Group != nil) || (c.Group != nil && group != *c.Group) {
			t.Errorf("Find(%q) = %q, %v, want %v", c.Path, group, ok, c.Group)
		}
	}
}
