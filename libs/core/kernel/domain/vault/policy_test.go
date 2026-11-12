package vault

import (
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestGetSourceKindClassifiesFilesByExtension(t *testing.T) {
	cases := []struct {
		name string
		want domain.SourceKind
	}{
		{"notes/today.md", domain.KindNote},
		{"notes/TODAY.MD", domain.KindNote},
		{"library/A Book.sample", domain.KindFile},
		{"pictures/cover", domain.KindFile},
	}
	for _, c := range cases {
		if kind := GetSourceKind(c.name); kind != c.want {
			t.Errorf("%s is %q, want %q", c.name, kind, c.want)
		}
	}
}
