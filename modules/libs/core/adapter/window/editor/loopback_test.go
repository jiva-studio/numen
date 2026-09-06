package editor

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
)

// heldVaults is the vaults an installation holds, for an address naming one the
// window is not showing.
type heldVaults []domain.Vault

func (l heldVaults) All() ([]domain.Vault, error) { return l, nil }

func (l heldVaults) Find(id string) (domain.Vault, bool, error) {
	for _, v := range l {
		if string(v.ID) == id {
			return v, true, nil
		}
	}
	return domain.Vault{}, false, nil
}

func (l heldVaults) Last() (domain.Vault, bool, error) { return domain.Vault{}, false, nil }

func (heldVaults) Save(domain.Vault) error { return nil }

func (heldVaults) Remove(domain.VaultID) error { return nil }

func (heldVaults) Opened(domain.VaultID) error { return nil }

// played is the socket a player reaches this API over, closed with the test.
func played(t *testing.T, api *API) (*Loopback, http.Handler) {
	t.Helper()
	back, err := Listen(api, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { back.Close() })
	return back, back.serving()
}

// statOf is what the vault says about the file at a path, which is what an
// address names. A path the vault does not hold has an address all the same,
// and what that address is answered with is what some of these ask about.
func statOf(t *testing.T, api *API, v domain.Vault, path string) domain.Fingerprint {
	t.Helper()
	reader, err := api.Readers.Open(v)
	if err != nil {
		return domain.Fingerprint{Path: path}
	}
	ref, err := reader.Stat(t.Context(), path)
	if err != nil {
		return domain.Fingerprint{Path: path}
	}
	return ref
}

// takenAway is a socket that stops answering the moment it is asked to, as one
// does where the machine takes it away.
type takenAway struct {
	net.Listener
	why error
}

func (t takenAway) Accept() (net.Conn, error) { return nil, t.why }

func (takenAway) Close() error { return nil }

func (takenAway) Addr() net.Addr { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)} }

// A socket that stopped answering is said, and gives out no more addresses. An
// address on a dead socket is a recording that will not play, with nothing
// anywhere to say why.
func TestASocketThatStoppedAnsweringIsSaid(t *testing.T) {
	api, _ := listeningTo(t, nil)

	said := make(chan error, 1)
	why := errors.New("the machine took the socket away")
	back, err := answering(takenAway{why: why}, api, func(err error) { said <- err })
	if err != nil {
		t.Fatal(err)
	}
	defer back.Close()

	select {
	case told := <-said:
		if !errors.Is(told, why) {
			t.Errorf("the socket stopping was said as %v", told)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the socket stopped answering and nobody was told")
	}
	if at := back.Address(api.Showing(), statOf(t, api, api.Showing(), talk)); at != "" {
		t.Errorf("a socket that answers nothing gave out %s", at)
	}
}

// The address is the whole of what tells this window's own asking from anybody
// else's, so a request without it is answered with nothing.
func TestOnlyTheAddressThisRunGaveOutIsAnswered(t *testing.T) {
	api, _ := listeningTo(t, nil)
	back, handler := played(t, api)

	right := back.Address(api.Showing(), statOf(t, api, api.Showing(), talk))
	if out := ask(handler, right); out.Code != http.StatusOK {
		t.Fatalf("the address this run gave out was answered %d", out.Code)
	}

	for _, wrong := range []string{
		"/files/" + talk,
		"/" + strings.Repeat("a", 32) + "/" + talk,
		"/",
	} {
		if out := ask(handler, wrong); out.Code != http.StatusForbidden {
			t.Errorf("%s was answered %d, want %d", wrong, out.Code, http.StatusForbidden)
		}
	}
}

// A tab plays on while the window is moved to another vault, so the vault is
// named in the address and not taken from the window.
func TestAFileIsServedFromTheVaultItsAddressNames(t *testing.T) {
	api, _ := listeningTo(t, nil)
	held := api.Showing()
	api.Vaults.Registry = heldVaults{held}
	back, handler := played(t, api)

	// The window moves to a vault holding nothing of the kind.
	api.show(testsupport.NewVault(t, map[string]string{"other.md": "somewhere else"}))

	if out := ask(handler, back.Address(held, statOf(t, api, held, talk))); out.Code != http.StatusOK {
		t.Errorf("the recording of the vault that moved out of the window was answered %d: %s",
			out.Code, out.Body)
	}
}

// A vault this installation does not hold is not a vault to read from.
func TestAFileOfAVaultNobodyHoldsIsRefused(t *testing.T) {
	api, _ := listeningTo(t, nil)
	back, handler := played(t, api)

	elsewhere := api.Showing()
	ref := statOf(t, api, elsewhere, talk)
	elsewhere.ID = "01ANOTHERVAULTALTOGETHER00"
	if out := ask(handler, back.Address(elsewhere, ref)); out.Code != http.StatusNotFound {
		t.Errorf("a vault nobody holds was answered %d", out.Code)
	}
}

// Every file of the vault is served, and a path leaving it is refused by the
// vault's own reader.
func TestTheSocketServesTheVaultAndNotTheDisk(t *testing.T) {
	vault := testsupport.NewVault(t, map[string]string{
		talk:      sound,
		"note.md": "# What a note says",
	})
	api := &API{Readers: filesystem.VaultReaders{}}
	api.show(vault)
	back, handler := played(t, api)

	if out := ask(handler, back.Address(vault, statOf(t, api, vault, "note.md"))); out.Code != http.StatusOK {
		t.Errorf("a note of the vault was answered %d", out.Code)
	}
	for _, outside := range []string{"../secrets", "/etc/passwd"} {
		at := back.Address(vault, statOf(t, api, vault, outside))
		if out := ask(handler, at); out.Code == http.StatusOK {
			t.Errorf("%s was served", outside)
		}
	}
}

// A player asks for a piece and is answered with that piece.
func TestTheSocketAnswersAPiece(t *testing.T) {
	api, _ := listeningTo(t, nil)
	back, handler := played(t, api)

	out := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, back.Address(api.Showing(), statOf(t, api, api.Showing(), talk)), nil)
	r.Header.Set("Range", "bytes=4-12")
	handler.ServeHTTP(out, r)

	if out.Code != http.StatusPartialContent {
		t.Fatalf("asked for a piece and got %d", out.Code)
	}
	if got := out.Body.String(); got != sound[4:13] {
		t.Errorf("the piece came back as %q", got)
	}
}

// The player reaches it over a socket, not through this process, so the whole
// way in is what is asked here.
func TestARecordingIsReachedOverTheSocket(t *testing.T) {
	api, _ := listeningTo(t, nil)
	back, err := Listen(api, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer back.Close()

	out, err := http.Get(back.Address(api.Showing(), statOf(t, api, api.Showing(), talk)))
	if err != nil {
		t.Fatalf("the socket answered nothing: %v", err)
	}
	defer out.Body.Close()
	if out.StatusCode != http.StatusOK {
		t.Fatalf("the socket answered %d", out.StatusCode)
	}
	if got := out.Header.Get("Content-Type"); got != "audio/mpeg" {
		t.Errorf("it came back as %q", got)
	}
	said, err := io.ReadAll(out.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(said) != sound {
		t.Errorf("it came back as %q", said)
	}
}

// A page is held to loading nothing but what its own handler serves, and the
// socket a recording is played from is the one thing named beside it.
func TestThePolicyNamesTheSocketAndNothingElse(t *testing.T) {
	api, _ := listeningTo(t, nil)
	back, err := Listen(api, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer back.Close()
	api.Playing = back

	said := handed(api.Serving(http.NotFoundHandler()), "/").
		Header().Get("Content-Security-Policy")
	if !strings.Contains(said, "media-src 'self' "+back.address) {
		t.Errorf("the socket is not what a recording may be played from: %q", said)
	}
	if strings.Contains(said, "img-src 'self' "+back.address) {
		t.Error("naming where a recording plays from widened where a picture comes from")
	}
}
