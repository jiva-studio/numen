package service

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/api"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/internal/testsupport"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/usecase/extract"
)

// TestMain makes the folder the pools of the tests keep the compiled PDFium in.
func TestMain(m *testing.M) {
	os.Exit(testsupport.Run(m))
}

func TestGetHighlightsPlacesRunsOfTheLiveVersion(t *testing.T) {
	texts, writer, _ := testsupport.NewVault(t, "tiny.pdf", "tiny.pdf")
	testsupport.CreateText(t, writer, extract.Request{Path: "tiny.pdf"})
	documents := NewDocumentServer(newReader(t, hostVault(t, "tiny.pdf"), texts))
	text := testsupport.ReadLive(t, texts, "tiny.pdf")
	beta, zeta := strings.Index(text.Text, "beta"), strings.Index(text.Text, "zeta")

	res, err := documents.GetHighlights(t.Context(), api.GetHighlightsRequest{
		Path: "tiny.pdf",
		Ranges: []api.Range{
			{Start: zeta, End: zeta + len("zeta")},
			{Start: beta, End: zeta + len("zeta")},
		},
	})
	if err != nil {
		t.Fatalf("GetHighlights: %v", err)
	}

	runs := res.Runs
	if len(runs) != 2 {
		t.Fatalf("%d runs, want the 2 asked about", len(runs))
	}
	if places := runs[0].Places; len(places) != 1 || places[0].Page != 1 || len(places[0].Rects) != 1 {
		t.Errorf("zeta is placed at %v, want one rectangle on page 1", places)
	}
	places := runs[1].Places
	if len(places) != 2 || places[0].Page != 0 || places[1].Page != 1 {
		t.Fatalf("beta to zeta is placed at %v, want pages 0 and 1", places)
	}
	if len(places[0].Rects) != 2 || len(places[1].Rects) != 3 {
		t.Errorf("beta to zeta covers %d and %d words, want 2 and 3", len(places[0].Rects), len(places[1].Rects))
	}
}

func TestGetHighlightsPlacesNothingWithoutLiveVersion(t *testing.T) {
	texts, _, _ := testsupport.NewVault(t, "tiny.pdf", "tiny.pdf")
	documents := NewDocumentServer(newReader(t, hostVault(t, "tiny.pdf"), texts))

	res, err := documents.GetHighlights(t.Context(), api.GetHighlightsRequest{
		Path: "tiny.pdf", Ranges: []api.Range{{Start: 0, End: 5}},
	})
	if err != nil {
		t.Fatalf("GetHighlights: %v", err)
	}

	if runs := res.Runs; len(runs) != 1 || len(runs[0].Places) != 0 {
		t.Errorf("runs %v, want one run with no places", runs)
	}
}

func TestGetHighlightsRefusesARangeThatIsNotOne(t *testing.T) {
	documents := NewDocumentServer(newReader(t, hostVault(t, "tiny.pdf"), hosttest.NewTexts("vault", testsupport.PluginID)))

	_, err := documents.GetHighlights(t.Context(), api.GetHighlightsRequest{
		Path: "tiny.pdf", Ranges: []api.Range{{Start: 4, End: 4}},
	})

	if codeOf(err) != server.CodeInvalidArgument {
		t.Errorf("GetHighlights = %v, want invalid_argument", err)
	}
}

func TestTheStatusOfAVaultFailure(t *testing.T) {
	for err, want := range map[error]int{
		host.ErrNoVault:  http.StatusForbidden,
		host.ErrOutside:  http.StatusForbidden,
		host.ErrNotExist: http.StatusNotFound,
	} {
		if got := getStatus(err); got != want {
			t.Errorf("getStatus(%v) = %d, want %d", err, got, want)
		}
	}
}

func hostVault(t *testing.T, name string) *hosttest.Vault {
	t.Helper()
	return hosttest.OnDisk(t, "vault", map[string]string{name: string(testsupport.ReadFixture(t, name))})
}
