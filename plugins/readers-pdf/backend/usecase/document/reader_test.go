package document

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/internal/testsupport"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

// TestMain makes the folder the pools of the tests keep the compiled PDFium in.
func TestMain(m *testing.M) {
	os.Exit(testsupport.Run(m))
}

func newReader(t *testing.T, vault *hosttest.Vault) *Reader {
	t.Helper()
	return NewReader(
		func(id domain.VaultID) host.FileReader { return vault.With(id) },
		func(domain.VaultID) LiveTexts { return hosttest.NewTexts("vault", testsupport.PluginID) },
		testsupport.NewWorkers(t))
}

type closingContext struct {
	context.Context
	workers *pdf.Workers
}

func (c closingContext) Err() error {
	if err := c.workers.Close(); err != nil {
		return err
	}
	return context.Canceled
}

func TestDrawPageDrawsNothingForARequestThatWasGivenUp(t *testing.T) {
	if MostPixels != 16_777_216 {
		t.Fatalf("MostPixels = %d, want 16777216", MostPixels)
	}
	vault := hosttest.CreateOnDisk(t, "vault", map[string]string{"broken.pdf": "not a pdf"})
	reader := newReader(t, vault)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	encoded, err := reader.drawPage(ctx, testsupport.ReadFixture(t, "tiny.pdf"), 0, 400)

	if !errors.Is(err, context.Canceled) || len(encoded) != 0 {
		t.Errorf("drew %d bytes, error %v", len(encoded), err)
	}

	_, renderErr := reader.RenderPage(t.Context(), "", "broken.pdf", 0, 400)
	if failure, ok := errors.AsType[*RenderError](renderErr); !ok || !errors.Is(failure, pdf.ErrNotPDF) {
		t.Errorf("RenderPage on broken.pdf = %v, want RenderError wrapping ErrNotPDF", renderErr)
	}

	_, closeJoinErr := reader.drawPage(closingContext{Context: t.Context(), workers: reader.workers}, testsupport.ReadFixture(t, "tiny.pdf"), 0, 400)
	joined, ok := closeJoinErr.(interface{ Unwrap() []error })
	if !errors.Is(closeJoinErr, context.Canceled) || !ok || len(joined.Unwrap()) != 2 || joined.Unwrap()[1] == nil {
		t.Errorf("drawPage when worker closes = %v, want Canceled joined with the close error", closeJoinErr)
	}
}

func TestGetPagesListsTheSizeOfEachPage(t *testing.T) {
	vault := hosttest.CreateOnDisk(t, "vault", map[string]string{"tiny.pdf": string(testsupport.ReadFixture(t, "tiny.pdf"))})

	pages, err := newReader(t, vault).GetPages(t.Context(), "", "tiny.pdf")

	if err != nil || len(pages) != 2 || pages[0].Width <= 0 || pages[0].Height <= 0 {
		t.Errorf("pages %+v, error %v, want the two pages of the file with their sizes", pages, err)
	}
}

func TestGetPagesNamesTheFailureToOpenAFile(t *testing.T) {
	vault := hosttest.CreateOnDisk(t, "vault", map[string]string{"broken.pdf": "not a pdf"})
	reader := newReader(t, vault)

	_, err := reader.GetPages(t.Context(), "", "broken.pdf")
	if !errors.Is(err, ErrOpen) || !errors.Is(err, pdf.ErrNotPDF) {
		t.Errorf("error %v, want ErrOpen wrapping ErrNotPDF", err)
	}

	_, missingErr := reader.GetPages(t.Context(), "", "missing.pdf")
	if !errors.Is(missingErr, host.ErrNotExist) {
		t.Errorf("missing error %v, want ErrNotExist", missingErr)
	}
}

func TestGetHighlightsRefusesRangesThatAreNotRangesOfAText(t *testing.T) {
	vault := hosttest.CreateOnDisk(t, "vault", map[string]string{"tiny.pdf": string(testsupport.ReadFixture(t, "tiny.pdf"))})
	reader := newReader(t, vault)

	tooMany := make([]domain.ByteRange, mostSpans+1)
	for i := range tooMany {
		tooMany[i] = domain.ByteRange{Start: 0, End: 1}
	}
	for name, ranges := range map[string][]domain.ByteRange{
		"none":      nil,
		"too many":  tooMany,
		"empty":     {{Start: 3, End: 3}},
		"negative":  {{Start: -1, End: 3}},
		"backwards": {{Start: 5, End: 3}},
	} {
		end, checkErr := checkRanges(ranges)
		if checkErr == nil || end != 0 {
			t.Errorf("%s checkRanges = (%d, %v), want (0, error)", name, end, checkErr)
		}
		_, err := reader.GetHighlights(t.Context(), "", "tiny.pdf", ranges)
		if failure, ok := errors.AsType[*RangeError](err); !ok || failure == nil {
			t.Errorf("%s: error %v, want a RangeError", name, err)
		}
	}

	maxValid := make([]domain.ByteRange, mostSpans)
	for i := range maxValid {
		maxValid[i] = domain.ByteRange{Start: 0, End: 5}
	}
	places, err := reader.GetHighlights(t.Context(), "", "tiny.pdf", maxValid)
	if err != nil || len(places) != mostSpans {
		t.Errorf("GetHighlights(mostSpans) = %d places, %v; want %d, nil", len(places), err, mostSpans)
	}

	if _, err := reader.GetHighlights(t.Context(), "", "../outside.pdf", maxValid[:1]); !errors.Is(err, host.ErrOutside) {
		t.Errorf("GetHighlights outside vault = %v, want ErrOutside", err)
	}
	hidden := hosttest.New("vault")
	hidden.Hide()
	noVaultReader := newReader(t, hidden)
	if _, err := noVaultReader.GetHighlights(t.Context(), "", "tiny.pdf", maxValid[:1]); !errors.Is(err, host.ErrNoVault) {
		t.Errorf("GetHighlights without vault = %v, want ErrNoVault", err)
	}
}
