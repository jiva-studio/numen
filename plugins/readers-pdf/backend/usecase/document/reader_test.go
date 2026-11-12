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

func TestDrawPageDrawsNothingForARequestThatWasGivenUp(t *testing.T) {
	reader := newReader(t, hosttest.New("vault"))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	encoded, err := reader.drawPage(ctx, testsupport.ReadFixture(t, "tiny.pdf"), 0, 400)

	if !errors.Is(err, context.Canceled) || len(encoded) != 0 {
		t.Errorf("drew %d bytes, error %v", len(encoded), err)
	}
}

func TestGetPagesListsTheSizeOfEachPage(t *testing.T) {
	vault := hosttest.OnDisk(t, "vault", map[string]string{"tiny.pdf": string(testsupport.ReadFixture(t, "tiny.pdf"))})

	pages, err := newReader(t, vault).GetPages(t.Context(), "", "tiny.pdf")

	if err != nil || len(pages) != 2 || pages[0].Width <= 0 || pages[0].Height <= 0 {
		t.Errorf("pages %+v, error %v, want the two pages of the file with their sizes", pages, err)
	}
}

func TestGetPagesNamesTheFailureToOpenAFile(t *testing.T) {
	vault := hosttest.OnDisk(t, "vault", map[string]string{"broken.pdf": "not a pdf"})

	_, err := newReader(t, vault).GetPages(t.Context(), "", "broken.pdf")

	if !errors.Is(err, ErrOpen) {
		t.Errorf("error %v, want ErrOpen", err)
	}
}

func TestGetHighlightsRefusesRangesThatAreNotRangesOfAText(t *testing.T) {
	vault := hosttest.OnDisk(t, "vault", map[string]string{"tiny.pdf": string(testsupport.ReadFixture(t, "tiny.pdf"))})
	reader := newReader(t, vault)

	for name, ranges := range map[string][]domain.ByteRange{
		"none":      nil,
		"too many":  make([]domain.ByteRange, mostSpans+1),
		"empty":     {{Start: 3, End: 3}},
		"negative":  {{Start: -1, End: 3}},
		"backwards": {{Start: 5, End: 3}},
	} {
		_, err := reader.GetHighlights(t.Context(), "", "tiny.pdf", ranges)
		if failure, ok := errors.AsType[*RangeError](err); !ok || failure == nil {
			t.Errorf("%s: error %v, want a RangeError", name, err)
		}
	}
}
