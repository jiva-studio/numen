package link

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
)

func dropAll(t *testing.T, l *Links, urls ...string) []string {
	t.Helper()
	paths, failed := l.CreateLinks(t.Context(), "talks", urls)
	l.Close()
	if len(failed) > 0 {
		t.Fatalf("failed = %v", failed)
	}
	return paths
}

func TestPageTitleNamesTheNote(t *testing.T) {
	l, _, _, _, _, _, sf, _ := createTestLinks()
	sf.meta = fetch.Metadata{Title: "Tidal mills"}

	paths := dropAll(t, l, "https://mills.test/a")

	if want := []string{"talks/Tidal mills.md"}; !slices.Equal(paths, want) {
		t.Errorf("paths = %v, want %v", paths, want)
	}
}

func TestSameTitleGetsSuffixes(t *testing.T) {
	l, _, _, _, _, _, sf, _ := createTestLinks()
	sf.meta = fetch.Metadata{Title: "Tidal mills"}

	paths := dropAll(t, l, "https://mills.test/a", "https://mills.test/b", "https://mills.test/c")

	want := []string{"talks/Tidal mills.md", "talks/Tidal mills 1.md", "talks/Tidal mills 2.md"}
	if !slices.Equal(paths, want) {
		t.Errorf("paths = %v, want %v", paths, want)
	}
}

func TestNoTitleFallsBackToTheAddress(t *testing.T) {
	l, _, _, _, _, _, sf, _ := createTestLinks()
	sf.metaErr = errors.New("503")
	name, _, err := domain.CreateFilename("https://mills.test/a")
	if err != nil {
		t.Fatal(err)
	}

	paths := dropAll(t, l, "https://mills.test/a")

	if want := []string{"talks/" + name + domain.NoteExtension}; !slices.Equal(paths, want) {
		t.Errorf("paths = %v, want %v", paths, want)
	}
}

func TestTitleIsSanitized(t *testing.T) {
	l, v, _, _, _, _, sf, _ := createTestLinks()
	sf.meta = fetch.Metadata{Title: "Mills: wind/water?"}

	paths := dropAll(t, l, "https://mills.test/a")

	if len(paths) != 1 || strings.Count(paths[0], "/") != 1 {
		t.Fatalf("paths = %v, want one note directly in talks", paths)
	}
	content, _ := v.ReadContent(paths[0])
	if !strings.Contains(content, "Mills: wind/water?") {
		t.Errorf("note = %q, want the page title in its frontmatter", content)
	}
}
