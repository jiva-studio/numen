package hosttest_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
)

// A proofreader reads the live version another plugin made, and puts its own made from it.
func TestAProofreaderReadsTheLiveVersionAndPutsItsOwn(t *testing.T) {
	ctx := t.Context()
	file := []byte("%PDF")
	ocr := hosttest.NewTexts("v1", "org.example.ocr")
	ocr.File("a.pdf", file)
	recognised := hosttest.NewMemoryProvider()
	recognised.Hold("a.pdf", "k", []domain.TextPart{{Text: "teh "}, {Text: "end", Place: domain.TextPlace{Page: 1}}})
	ocr.Provide("org.example.ocr", recognised)
	a, err := ocr.PutVersion(ctx, host.TextPut{Path: "a.pdf", Key: "k", Format: domain.TextFormatPositioned, InputHash: domain.HashContent(file), Hash: "ha"})
	if err != nil {
		t.Fatal(err)
	}

	fixer := ocr.As("org.example.fixer")
	listing, err := fixer.ListVersions(ctx, "a.pdf")
	if err != nil || listing.Live != a.GetID() {
		t.Fatalf("live %q (%v), want %q", listing.Live, err, a.GetID())
	}
	text, err := host.ReadWholeText(ctx, fixer, listing.Versions[0])
	if err != nil || text.Text != "teh end" {
		t.Fatalf("read %q (%v)", text.Text, err)
	}
	b, err := fixer.PutVersion(ctx, host.TextPut{Path: "a.pdf", Key: "b", Input: a.GetID(), Format: domain.TextFormatPositioned, InputHash: a.Hash})
	if err != nil {
		t.Fatal(err)
	}
	if listing, _ := fixer.ListVersions(ctx, "a.pdf"); listing.Live != b.GetID() {
		t.Fatalf("live %q, want the proofreading", listing.Live)
	}

	ocr.SetEnabled("org.example.ocr", false)
	if listing, _ := fixer.ListVersions(ctx, "a.pdf"); listing.Live != "" {
		t.Fatalf("live %q with the recogniser off, want none", listing.Live)
	}
	if got := ocr.TakeChanges(); !slices.Equal(got, []string{"a.pdf", "a.pdf"}) {
		t.Fatalf("changes %v", got)
	}
}

func TestAVersionThatIsNotOneIsRefused(t *testing.T) {
	texts := hosttest.NewTexts("v1", "org.example.ocr")
	for name, put := range map[string]host.TextPut{
		"no key":             {Path: "a.pdf", Format: domain.TextFormatPlain},
		"no format":          {Path: "a.pdf", Key: "k"},
		"an input not there": {Path: "a.pdf", Key: "k", Format: domain.TextFormatPlain, Input: "x/y"},
	} {
		if _, err := texts.PutVersion(t.Context(), put); !errors.Is(err, host.ErrInvalidVersion) {
			t.Errorf("%s: got %v", name, err)
		}
	}
	if _, err := texts.ReadText(t.Context(), "a.pdf", "x/y", ""); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("a version not there: got %v", err)
	}
}
