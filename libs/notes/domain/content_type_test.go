package domain_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestParseContentTypeLowersTheType(t *testing.T) {
	got, err := domain.ParseContentType("Application/PDF")
	if err != nil || got != "application/pdf" {
		t.Fatalf("ParseContentType = %q, %v; want application/pdf", got, err)
	}
}

func TestParseContentTypeAcceptsMasks(t *testing.T) {
	for _, text := range []string{"audio/*", "*/*", "application/epub+zip", "application/vnd.ms-excel"} {
		if _, err := domain.ParseContentType(text); err != nil {
			t.Errorf("ParseContentType(%q) = %v", text, err)
		}
	}
}

func TestParseContentTypeRejectsNonTypes(t *testing.T) {
	for _, text := range []string{"", "pdf", ".pdf", "*/pdf", "audio/", "/mpeg", "audio/mp*", "a/b/c"} {
		if _, err := domain.ParseContentType(text); !errors.Is(err, domain.ErrInvalidContentType) {
			t.Errorf("ParseContentType(%q) = %v; want ErrInvalidContentType", text, err)
		}
	}
}

func TestContentTypeMatches(t *testing.T) {
	cases := []struct {
		mask, of domain.ContentType
		want     bool
	}{
		{"audio/*", "audio/mpeg", true},
		{"audio/*", "video/mp4", false},
		{"audio/*", "audio/*", false},
		{"*/*", domain.UnknownContentType, true},
		{"application/pdf", "application/pdf", true},
		{"application/pdf", "application/pdfx", false},
		{"application/*", "applications/pdf", false},
	}
	for _, c := range cases {
		if got := c.mask.Matches(c.of); got != c.want {
			t.Errorf("%q.Matches(%q) = %v; want %v", c.mask, c.of, got, c.want)
		}
	}
}

func TestChooseHandlerPrefersTheNarrowerOfTwoDefaults(t *testing.T) {
	candidates := []domain.ContentHandler{
		{PluginID: "a.plugin", ID: "player", Type: "audio/*", Rank: domain.RankDefault},
		{PluginID: "z.plugin", ID: "mp3", Type: "audio/mpeg", Rank: domain.RankDefault},
	}
	got, ok := domain.ChooseHandler("audio/mpeg", candidates)
	if !ok || got.PluginID != "z.plugin" {
		t.Fatalf("ChooseHandler = %+v, %v; want z.plugin", got, ok)
	}
}

func TestChooseHandlerPrefersTheLowerPluginIDOfTwoDefaultsOfOneType(t *testing.T) {
	candidates := []domain.ContentHandler{
		{PluginID: "org.b.reader", ID: "pdf", Type: "application/pdf", Rank: domain.RankDefault},
		{PluginID: "org.a.reader", ID: "pdf", Type: "application/pdf", Rank: domain.RankDefault},
	}
	for range 2 {
		got, ok := domain.ChooseHandler("application/pdf", candidates)
		if !ok || got.PluginID != "org.a.reader" {
			t.Fatalf("ChooseHandler = %+v, %v; want org.a.reader", got, ok)
		}
		candidates[0], candidates[1] = candidates[1], candidates[0]
	}
}

func TestChooseHandlerSkipsAlternates(t *testing.T) {
	candidates := []domain.ContentHandler{
		{PluginID: "a.plugin", ID: "raw", Type: "application/pdf", Rank: domain.RankAlternate},
		{PluginID: "b.plugin", ID: "any", Type: "*/*", Rank: domain.RankDefault},
	}
	got, ok := domain.ChooseHandler("application/pdf", candidates)
	if !ok || got.PluginID != "b.plugin" {
		t.Fatalf("ChooseHandler = %+v, %v; want b.plugin", got, ok)
	}
	if _, ok := domain.ChooseHandler("application/pdf", candidates[:1]); ok {
		t.Fatal("ChooseHandler chose an alternate handler")
	}
}

func TestChooseHandlerFindsNoneForAnotherType(t *testing.T) {
	candidates := []domain.ContentHandler{{PluginID: "a.plugin", ID: "pdf", Type: "application/pdf", Rank: domain.RankDefault}}
	if got, ok := domain.ChooseHandler("application/epub+zip", candidates); ok {
		t.Fatalf("ChooseHandler = %+v; want none", got)
	}
}
