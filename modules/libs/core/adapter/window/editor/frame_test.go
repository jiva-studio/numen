package editor

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

const pointsAtAVideo = "https://youtu.be/dQw4w9WgXcQ"

// A host checks which page frames its player, and it is told by the address
// that page was loaded from. A window drawn from its own scheme has none to
// give — a browser sends no such address for a scheme that is not http — so the
// page holding the player is served over this run's socket, and the host is
// told an address on this machine.
func TestThePlayerIsFramedFromThisRunsOwnSocket(t *testing.T) {
	back, handler := openLoopback(t, newBareServices())
	pos := parseURL(t, pointsAtAVideo)

	address := back.Embed(pos)
	if !strings.HasPrefix(address, back.address+"/") {
		t.Fatalf("the player is framed from %q, want this run's socket", address)
	}

	answer := httptest.NewRecorder()
	handler.ServeHTTP(answer, httptest.NewRequest(http.MethodGet, address, nil))
	if answer.Code != http.StatusOK {
		t.Fatalf("the page answered %d: %s", answer.Code, answer.Body.String())
	}
	page := answer.Body.String()
	if !strings.Contains(page, "origin="+url.QueryEscape(back.address)) {
		t.Errorf("the host is told nothing about who frames it: %s", page)
	}
	if !strings.Contains(page, getPlayerURL(pos)) {
		t.Errorf("the page frames %q, want the player of %q", page, string(pos))
	}
}

// The route carries the address the file points at, and what plays it is this
// adapter's to say. A host learned about later is played here without this
// route learning anything about it.
func TestTheRouteCarriesTheAddressAndNotOneHostsIdentifier(t *testing.T) {
	back, _ := openLoopback(t, newBareServices())
	pos := parseURL(t, pointsAtAVideo)

	address := back.Embed(pos)

	if !strings.Contains(address, url.PathEscape(string(pos))) {
		t.Errorf("the route reads %q, want the address it points at", address)
	}
	if strings.Contains(address, "/dQw4w9WgXcQ") {
		t.Errorf("the route names one host's identifier: %q", address)
	}
}

// That page frames the host and reaches nowhere else. It is served over a
// socket every process on this machine can knock at, so what it may do is
// written on it.
func TestThePageContainingAPlayerIsHeldToItsOwnPolicy(t *testing.T) {
	back, handler := openLoopback(t, newBareServices())

	answer := httptest.NewRecorder()
	handler.ServeHTTP(answer,
		httptest.NewRequest(http.MethodGet, back.Embed(parseURL(t, pointsAtAVideo)), nil))

	response := answer.Header().Get("Content-Security-Policy")
	if !strings.Contains(response, "default-src 'none'") {
		t.Errorf("the page may reach anywhere: %q", response)
	}
	if !strings.Contains(response, "frame-src "+embedHosts[0]) {
		t.Errorf("the page frames %q", response)
	}
}

// way to ask this socket for a page of somebody else's.
func TestAnEmbedAddressNothingPlays(t *testing.T) {
	back, handler := openLoopback(t, newBareServices())

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

// parseURL is one address a note points at.
func parseURL(t *testing.T, output string) domain.URL {
	t.Helper()
	pos, err := domain.ParseURL(output)
	if err != nil {
		t.Fatal(err)
	}
	return pos
}
