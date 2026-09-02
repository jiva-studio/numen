package webui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
)

// heldVaults is the vaults an installation holds, for an address naming one the
// window is not showing.
type heldVaults []domain.Vault

func (l heldVaults) All() ([]domain.Vault, error) { return l, nil }

func (l heldVaults) Find(string) (domain.Vault, bool, error) { return domain.Vault{}, false, nil }

func (l heldVaults) Last() (domain.Vault, bool, error) { return domain.Vault{}, false, nil }

func (heldVaults) Save(domain.Vault) error { return nil }

func (heldVaults) Remove(string) error { return nil }

func (heldVaults) Opened(string) error { return nil }

// played is the socket a player reaches this API over, closed with the test.
func played(t *testing.T, api *API) (*Loopback, http.Handler) {
	t.Helper()
	back, err := Listen(api)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { back.Close() })
	return back, back.serving()
}

// The address is the whole of what tells this window's own asking from anybody
// else's, so a request without it is answered with nothing.
func TestOnlyTheAddressThisRunGaveOutIsAnswered(t *testing.T) {
	api, _ := listeningTo(t, nil)
	back, handler := played(t, api)

	right := back.Address(api.Showing(), talk)
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
	api.Vaults = heldVaults{held}
	back, handler := played(t, api)

	// The window moves to a vault holding nothing of the kind.
	api.show(testsupport.NewVault(t, map[string]string{"other.md": "somewhere else"}))

	if out := ask(handler, back.Address(held, talk)); out.Code != http.StatusOK {
		t.Errorf("the recording of the vault that moved out of the window was answered %d: %s",
			out.Code, out.Body)
	}
}

// A vault this installation does not hold is not a vault to read from.
func TestAFileOfAVaultNobodyHoldsIsRefused(t *testing.T) {
	api, _ := listeningTo(t, nil)
	back, handler := played(t, api)

	elsewhere := api.Showing()
	elsewhere.ID = "01ANOTHERVAULTALTOGETHER00"
	if out := ask(handler, back.Address(elsewhere, talk)); out.Code != http.StatusNotFound {
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
	api := &API{Readers: filesystem.Readers{}}
	api.show(vault)
	back, handler := played(t, api)

	if out := ask(handler, back.Address(vault, "note.md")); out.Code != http.StatusOK {
		t.Errorf("a note of the vault was answered %d", out.Code)
	}
	for _, outside := range []string{"../secrets", "/etc/passwd"} {
		if out := ask(handler, back.Address(vault, outside)); out.Code == http.StatusOK {
			t.Errorf("%s was served", outside)
		}
	}
}

// What a file is served as is read off its name, so a player is not left to
// guess from the bytes.
func TestWhatAFileIsServedAs(t *testing.T) {
	for name, want := range map[string]string{
		"talks/one.mp3":  "audio/mpeg",
		"talks/one.wav":  "audio/wav",
		"talks/one.flac": "audio/flac",
		"talks/one.md":   "",
	} {
		if got := servedAs(name); got != want {
			t.Errorf("%s is served as %q, want %q", name, got, want)
		}
	}
}

// A player asks for a piece and is answered with that piece.
func TestTheSocketAnswersAPiece(t *testing.T) {
	api, _ := listeningTo(t, nil)
	back, handler := played(t, api)

	out := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, back.Address(api.Showing(), talk), nil)
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
	back, err := Listen(api)
	if err != nil {
		t.Fatal(err)
	}
	defer back.Close()

	out, err := http.Get(back.Address(api.Showing(), talk))
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
	back, err := Listen(api)
	if err != nil {
		t.Fatal(err)
	}
	defer back.Close()
	api.Playing = back

	said := handed(api.Serving(http.NotFoundHandler()), "/").
		Header().Get("Content-Security-Policy")
	if !strings.Contains(said, "media-src 'self' "+back.at) {
		t.Errorf("the socket is not what a recording may be played from: %q", said)
	}
	if strings.Contains(said, "img-src 'self' "+back.at) {
		t.Error("naming where a recording plays from widened where a picture comes from")
	}
}
