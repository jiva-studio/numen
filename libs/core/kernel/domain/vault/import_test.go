package vault_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
)

func TestImportResult_DirectChildren(t *testing.T) {
	result := vault.ImportResult{Imported: []string{"Cover.png", "physics/Cover.png", "physics/deep/Slide.png"}}

	if got := result.DirectChildren(""); !slices.Equal(got, []string{"Cover.png"}) {
		t.Errorf("root children = %v", got)
	}
	if got := result.DirectChildren("physics"); !slices.Equal(got, []string{"physics/Cover.png"}) {
		t.Errorf("physics children = %v", got)
	}
}

func TestImportResult_DescribeFailures(t *testing.T) {
	why := errors.New("exists")
	cases := []struct {
		names []string
		want  string
	}{
		{nil, ""},
		{[]string{"a"}, "a: exists"},
		{[]string{"a", "b", "c", "d", "e"}, "a: exists; b: exists; c: exists; and 2 more"},
	}
	for _, c := range cases {
		var result vault.ImportResult
		for _, name := range c.names {
			result.Errors = append(result.Errors, vault.ImportFailure{Name: name, Why: why})
		}
		if got := result.DescribeFailures(); got != c.want {
			t.Errorf("DescribeFailures(%v) = %q, want %q", c.names, got, c.want)
		}
	}
}
