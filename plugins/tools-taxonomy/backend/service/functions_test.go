package service_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/searchfunction/searchfunctiontest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server/servertest"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/service"
	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/usecase/tagcache"
)

const functionToken = "tags-token"

// failingFrontmatter reads the frontmatter of the vault until isFailing is set.
type failingFrontmatter struct {
	tagcache.Frontmatter
	isFailing *atomic.Bool
}

func (f failingFrontmatter) ReadFrontmatter(ctx context.Context, q host.FrontmatterQuery) (host.FrontmatterPage, error) {
	if f.isFailing.Load() {
		return host.FrontmatterPage{}, errors.New("the host is not answering")
	}
	return f.Frontmatter.ReadFrontmatter(ctx, q)
}

type functionScene struct {
	vault     *hosttest.Vault
	isFailing *atomic.Bool
	client    *searchfunctiontest.Client
}

func newFunctionScene(t *testing.T, files map[string]string) functionScene {
	t.Helper()
	v := hosttest.New("garden")
	for path, content := range files {
		v.WriteFile(path, content)
	}
	isFailing := &atomic.Bool{}
	app, err := service.NewApp(func(id domain.VaultID) tagcache.Vault {
		return tagcache.Vault{Frontmatter: failingFrontmatter{Frontmatter: v.With(id), isFailing: isFailing}, Feed: v.With(id), Store: v.Store().With(id)}
	})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := servertest.NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": functionToken}, func(s *server.Server) error {
		app.Functions().Mount(s)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)
	return functionScene{vault: v, isFailing: isFailing, client: searchfunctiontest.NewClient(httpServer.Client(), httpServer.URL, functionToken)}
}

func getTagsVault() map[string]string {
	return map[string]string{
		"a.md": "---\ntags: [a]\n---\n",
		"b.md": "---\ntags: A/b\n---\n",
		"c.md": "---\ntags: '#a'\n---\n",
		"d.md": "---\ntags: Ab\n---\n",
		"e.md": "# no tags\n",
		"f.md": "---\ntags: [Враджа/Матхура]\n---\n",
	}
}

func callFunction(function, tag string) searchfunctiontest.Call {
	return searchfunctiontest.Call{Function: function, VaultID: "garden", IndexGeneration: "g4", Arguments: map[string]any{"tag": tag}}
}

func callTags(tag string) searchfunctiontest.Call {
	return callFunction("hasTag", tag)
}

func getPortVault() map[string]string {
	return map[string]string{
		"a.md": "---\ntags: Port\n---\n",
		"b.md": "---\ntags: port\n---\n",
		"c.md": "---\ntags: '#Port'\n---\n",
		"d.md": "---\ntags: [Port/Cargo]\n---\n",
		"e.md": "---\ntags: Portable\n---\n",
		"f.md": "# no tags\n",
	}
}

func getPortPaths() []string {
	return []string{"a.md", "b.md", "c.md", "d.md", "e.md", "f.md"}
}

func TestHasTagMatchesTheNotesWithTheTagOrANestedTagInAnyCase(t *testing.T) {
	s := newFunctionScene(t, getTagsVault())

	matched, warning, err := s.client.MatchDocuments(t.Context(), callTags("A"), []string{"a.md", "b.md", "c.md", "d.md", "e.md"})
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(matched, []string{"a.md", "b.md", "c.md"}) || warning != "" {
		t.Errorf("hasTag(\"A\") matched %v (%q), want a.md, b.md and c.md", matched, warning)
	}
	nested, _, err := s.client.MatchDocuments(t.Context(), callTags("враджа"), []string{"a.md", "f.md"})
	if err != nil || !slices.Equal(nested, []string{"f.md"}) {
		t.Errorf("hasTag(\"враджа\") matched %v, %v, want f.md", nested, err)
	}
}

func TestHasAnyTagFindsEveryNoteWithATagSortedWithTheExactCount(t *testing.T) {
	s := newFunctionScene(t, getTagsVault())

	paths, count, _, err := s.client.FindDocuments(t.Context(), searchfunctiontest.Call{Function: "hasAnyTag", VaultID: "garden"}, 3)
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(paths, []string{"a.md", "b.md", "c.md"}) || count != 5 {
		t.Errorf("hasAnyTag() found %v of %d, want the first three of five", paths, count)
	}
}

func TestHasTagWithoutATagIsInvalidArgument(t *testing.T) {
	s := newFunctionScene(t, getTagsVault())

	_, _, _, err := s.client.FindDocuments(t.Context(), callTags(" # "), 10)

	if err == nil || !strings.Contains(err.Error(), "invalid_argument") || !strings.Contains(err.Error(), "hasTag needs a tag") {
		t.Errorf("err = %v, want InvalidArgument", err)
	}
}

func TestTagsThatCannotBeUpdatedAnswerWithAWarningOfTheGeneration(t *testing.T) {
	s := newFunctionScene(t, getTagsVault())
	if _, _, _, err := s.client.FindDocuments(t.Context(), callTags("a"), 10); err != nil {
		t.Fatal(err)
	}
	s.isFailing.Store(true)
	s.vault.WriteFile("e.md", "---\ntags: a\n---\n")

	paths, _, warning, err := s.client.FindDocuments(t.Context(), callTags("a"), 10)
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(paths, []string{"a.md", "b.md", "c.md"}) || warning != "the tags may be out of date (index generation g4)" {
		t.Errorf("found %v with %q, want the kept tags and a warning", paths, warning)
	}
}

func TestHasTagMatchesANestedTagInAnyCase(t *testing.T) {
	s := newFunctionScene(t, getPortVault())

	matched, _, err := s.client.MatchDocuments(t.Context(), callFunction("hasTag", "port"), getPortPaths())

	if err != nil || !slices.Equal(matched, []string{"a.md", "b.md", "c.md", "d.md"}) {
		t.Errorf("hasTag(\"port\") matched %v, %v, want a.md, b.md, c.md and d.md", matched, err)
	}
}

func TestHasTagExactlyMatchesTheNotesWithThatTagAloneInAnyCaseWithOrWithoutTheHash(t *testing.T) {
	s := newFunctionScene(t, getPortVault())

	matched, warning, err := s.client.MatchDocuments(t.Context(), callFunction("hasTagExactly", "Port"), getPortPaths())

	if err != nil || warning != "" || !slices.Equal(matched, []string{"a.md", "b.md", "c.md"}) {
		t.Errorf("hasTagExactly(\"Port\") matched %v (%q), %v, want a.md, b.md and c.md", matched, warning, err)
	}
	hashed, _, err := s.client.MatchDocuments(t.Context(), callFunction("hasTagExactly", " #PORT "), getPortPaths())
	if err != nil || !slices.Equal(hashed, []string{"a.md", "b.md", "c.md"}) {
		t.Errorf("hasTagExactly(\" #PORT \") matched %v, %v, want a.md, b.md and c.md", hashed, err)
	}
}

func TestHasTagExactlyDoesNotMatchTheNotesWithOnlyANestedTagOrALongerName(t *testing.T) {
	s := newFunctionScene(t, getPortVault())

	matched, _, err := s.client.MatchDocuments(t.Context(), callFunction("hasTagExactly", "Port"), []string{"d.md", "e.md"})

	if err != nil || len(matched) != 0 {
		t.Errorf("hasTagExactly(\"Port\") matched %v, %v, want no note", matched, err)
	}
	nested, _, err := s.client.MatchDocuments(t.Context(), callFunction("hasTagExactly", "port/cargo"), getPortPaths())
	if err != nil || !slices.Equal(nested, []string{"d.md"}) {
		t.Errorf("hasTagExactly(\"port/cargo\") matched %v, %v, want d.md", nested, err)
	}
}

func TestHasTagExactlyFindsTheNotesWithThatTagSortedWithTheExactCount(t *testing.T) {
	s := newFunctionScene(t, getPortVault())

	paths, count, _, err := s.client.FindDocuments(t.Context(), callFunction("hasTagExactly", "Port"), 2)

	if err != nil || !slices.Equal(paths, []string{"a.md", "b.md"}) || count != 3 {
		t.Errorf("hasTagExactly(\"Port\") found %v of %d, %v, want the first two of three", paths, count, err)
	}
}

func TestHasTagAndHasTagExactlyMatchNoNoteWithoutTags(t *testing.T) {
	s := newFunctionScene(t, getPortVault())

	for _, function := range []string{"hasTag", "hasTagExactly"} {
		matched, _, err := s.client.MatchDocuments(t.Context(), callFunction(function, "Port"), []string{"f.md"})
		if err != nil || len(matched) != 0 {
			t.Errorf("%s(\"Port\") matched %v, %v, want no note", function, matched, err)
		}
	}
}

func TestHasTagExactlyWithoutATagIsInvalidArgument(t *testing.T) {
	s := newFunctionScene(t, getPortVault())

	_, _, _, err := s.client.FindDocuments(t.Context(), callFunction("hasTagExactly", " # "), 10)

	if err == nil || !strings.Contains(err.Error(), "invalid_argument") || !strings.Contains(err.Error(), "hasTagExactly needs a tag") {
		t.Errorf("err = %v, want InvalidArgument", err)
	}
}

func TestHasAnyTagMatchesTheNotesWithAnyTagAndNotThoseWithout(t *testing.T) {
	s := newFunctionScene(t, getPortVault())

	matched, _, err := s.client.MatchDocuments(t.Context(), searchfunctiontest.Call{Function: "hasAnyTag", VaultID: "garden"}, getPortPaths())

	if err != nil || !slices.Equal(matched, []string{"a.md", "b.md", "c.md", "d.md", "e.md"}) {
		t.Errorf("hasAnyTag() matched %v, %v, want every note but f.md", matched, err)
	}
}

func TestTheOldNameTagsIsAnUnknownFunction(t *testing.T) {
	s := newFunctionScene(t, getPortVault())

	_, _, _, err := s.client.FindDocuments(t.Context(), callFunction("tags", "Port"), 10)

	if err == nil || !strings.Contains(err.Error(), "not_found") {
		t.Errorf("err = %v, want NotFound", err)
	}
}

func TestTheOldNameIsTaggedIsAnUnknownFunction(t *testing.T) {
	s := newFunctionScene(t, getPortVault())

	_, _, _, err := s.client.FindDocuments(t.Context(), searchfunctiontest.Call{Function: "isTagged", VaultID: "garden"}, 10)

	if err == nil || !strings.Contains(err.Error(), "not_found") {
		t.Errorf("err = %v, want NotFound", err)
	}
}
