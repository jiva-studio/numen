package editor

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

const pointsAtAVideo = "https://youtu.be/dQw4w9WgXcQ"

// A host checks which page frames its player, and it is told by the address
// that page was loaded from. A window drawn from its own scheme has none to
// give — a browser sends no such address for a scheme that is not http — so the
// page holding the player is served over this run's socket, and the host is
// told an address on this machine.
func TestThePlayerIsFramedFromThisRunsOwnSocket(t *testing.T) {
	back, handler := played(t, &API{})
	at := pointing(t, pointsAtAVideo)

	address := back.Embed(at)
	if !strings.HasPrefix(address, back.address+"/") {
		t.Fatalf("the player is framed from %q, want this run's socket", address)
	}

	answer := httptest.NewRecorder()
	handler.ServeHTTP(answer, httptest.NewRequest(http.MethodGet, address, nil))
	if answer.Code != http.StatusOK {
		t.Fatalf("the page answered %d: %s", answer.Code, answer.Body.String())
	}
	page := answer.Body.String()
	// What the host is told is where the page holding it stands.
	if !strings.Contains(page, "origin="+url.QueryEscape(back.address)) {
		t.Errorf("the host is told nothing about who frames it: %s", page)
	}
	if !strings.Contains(page, at.Embed()) {
		t.Errorf("the page frames %q, want the player of %q", page, at.URL)
	}
}

// The route carries the address the note points at, and what plays it is the
// domain's to say. A host learned about later is played here without this route
// learning anything about it.
func TestTheRouteCarriesTheAddressAndNotOneHostsIdentifier(t *testing.T) {
	back, _ := played(t, &API{})
	at := pointing(t, pointsAtAVideo)

	address := back.Embed(at)

	if !strings.Contains(address, url.PathEscape(at.URL)) {
		t.Errorf("the route reads %q, want the address it points at", address)
	}
	if strings.Contains(address, "/"+at.Video) {
		t.Errorf("the route names one host's identifier: %q", address)
	}
}

// That page frames the host and reaches nowhere else. It is served over a
// socket every process on this machine can knock at, so what it may do is
// written on it.
func TestThePageHoldingAPlayerIsHeldToItsOwnPolicy(t *testing.T) {
	back, handler := played(t, &API{})

	answer := httptest.NewRecorder()
	handler.ServeHTTP(answer,
		httptest.NewRequest(http.MethodGet, back.Embed(pointing(t, pointsAtAVideo)), nil))

	said := answer.Header().Get("Content-Security-Policy")
	if !strings.Contains(said, "default-src 'none'") {
		t.Errorf("the page may reach anywhere: %q", said)
	}
	if !strings.Contains(said, "frame-src "+domain.EmbedHosts()[0]) {
		t.Errorf("the page frames %q", said)
	}
}

// An address nothing plays is refused. The route serves a player, and is not a
// way to ask this socket for a page of somebody else's.
func TestAnEmbedAddressNothingPlays(t *testing.T) {
	back, handler := played(t, &API{})

	for _, raw := range []string{
		url.PathEscape("https://example.com/entropy"),
		url.PathEscape("file:///etc/passwd"),
		"not-an-address",
	} {
		answer := httptest.NewRecorder()
		handler.ServeHTTP(answer, httptest.NewRequest(http.MethodGet,
			back.address+"/"+back.token+"/"+embedRoute+"/"+raw, nil))

		if answer.Code != http.StatusBadRequest {
			t.Errorf("%s answered %d: %s", raw, answer.Code, answer.Body.String())
		}
	}
}

// pointing is one address a note points at.
func pointing(t *testing.T, written string) domain.WebAddress {
	t.Helper()
	at, err := domain.ParseWebAddress(written)
	if err != nil {
		t.Fatal(err)
	}
	return at
}
