package hosttest_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

func TestTheFrontmatterOfTheNotesIsReadInPagesOfTheKeysAsked(t *testing.T) {
	v := hosttest.New("garden")
	v.WriteFile("c.md", "---\ntags: [river]\n---\nC\n")
	v.WriteFile("a.md", "---\ntags: lake\nstatus: open\n---\nA\n")
	v.WriteFile("b.md", "B\n")
	v.WriteFile("scan.pdf", "%PDF")

	first, err := v.ReadFrontmatter(t.Context(), host.FrontmatterQuery{Keys: []string{"tags"}, PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	second, err := v.ReadFrontmatter(t.Context(), host.FrontmatterQuery{Keys: []string{"tags"}, PageSize: 2, PageToken: first.NextPageToken})
	if err != nil {
		t.Fatal(err)
	}

	if len(first.Notes) != 2 || first.Notes[0].Path != "a.md" || first.Notes[0].Values["tags"] != "lake" ||
		first.Notes[0].Values["status"] != nil || first.Notes[1].Path != "b.md" || first.NextPageToken != "b.md" {
		t.Errorf("the first page is %+v", first)
	}
	if len(second.Notes) != 1 || second.Notes[0].Path != "c.md" || !reflect.DeepEqual(second.Notes[0].Values["tags"], []any{"river"}) ||
		second.NextPageToken != "" {
		t.Errorf("the second page is %+v", second)
	}
}

func TestTheFrontmatterOfNamedNotesOmitsAPathWithNoNote(t *testing.T) {
	v := hosttest.New("garden")
	v.WriteFile("a.md", "---\ntags: lake\n---\nA\n")

	page, err := v.ReadFrontmatter(t.Context(), host.FrontmatterQuery{Paths: []string{"gone.md", "a.md"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Notes) != 1 || page.Notes[0].Path != "a.md" {
		t.Errorf("read %+v, want a.md alone", page)
	}
	if _, err := v.ReadFrontmatter(t.Context(), host.FrontmatterQuery{Keys: make([]string, 65)}); !errors.Is(err, host.ErrInvalidFrontmatterQuery) {
		t.Errorf("65 keys answered %v", err)
	}
}
