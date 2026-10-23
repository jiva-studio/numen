package editor

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

const (
	talk  = "talks/lecture.mp3"
	sound = "the bytes of a recording, long enough to ask for a piece of"
)

func openRecordingWindow(t *testing.T, _ any) (*Services, http.Handler) {
	t.Helper()
	vault := testsupport.NewVault(t, map[string]string{talk: sound})
	api := newBareServices()
	api.readFiles(vaultreader.Readers{})
	api.state.setVault(vault)
	return api, api.NewHandler(http.NotFoundHandler())
}

// heldVaults is the vaults an installation holds, for an address naming one the
// window is not showing.
type heldVaults []domain.Vault

func (l heldVaults) List() ([]domain.Vault, error) { return l, nil }

func (l heldVaults) FindVault(id string) (domain.Vault, bool, error) {
	for _, v := range l {
		if string(v.ID) == id {
			return v, true, nil
		}
	}
	return domain.Vault{}, false, nil
}

func (l heldVaults) GetLastVault() (domain.Vault, bool, error) { return domain.Vault{}, false, nil }

func (heldVaults) Save(domain.Vault) error { return nil }

func (heldVaults) Remove(domain.VaultID) error { return nil }

func (heldVaults) RecordOpened(domain.VaultID) error { return nil }

// openLoopback is the socket a player reaches this API over, closed with the test.
func openLoopback(t *testing.T, api *Services) (*Loopback, http.Handler) {
	t.Helper()
	back, err := Listen(api.server, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { back.Close() })
	return back, back.getHandler()
}

func ask(handler http.Handler, url string) *httptest.ResponseRecorder {
	out := httptest.NewRecorder()
	handler.ServeHTTP(out, httptest.NewRequest(http.MethodGet, url, nil))
	return out
}

// statOf is what the vault says about the file at a path, which is what an
// address names. A path the vault does not hold has an address all the same,
// and what that address is answered with is what some of these ask about.
func statOf(t *testing.T, api *Services, v domain.Vault, path string) domain.Fingerprint {
	t.Helper()
	ref, err := api.files.getFingerprint.Execute(t.Context(), v, path)
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
	api, _ := openRecordingWindow(t, nil)

	response := make(chan error, 1)
	why := errors.New("the machine took the socket away")
	back, err := newLoopback(takenAway{why: why}, api.server, func(err error) { response <- err })
	if err != nil {
		t.Fatal(err)
	}
	defer back.Close()

	select {
	case res := <-response:
		if !errors.Is(res, why) {
			t.Errorf("the socket stopping was said as %v", res)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the socket stopped answering and nobody received")
	}
	if pos := back.Address(api.state.GetCurrentVault(), statOf(t, api, api.state.GetCurrentVault(), talk)); pos != "" {
		t.Errorf("a socket that answers nothing gave out %s", pos)
	}
}

// The address is the whole of what tells this window's own asking from anybody
// else's, so a request without it is answered with nothing.
func TestOnlyTheAddressThisRunGaveOutIsAnswered(t *testing.T) {
	api, _ := openRecordingWindow(t, nil)
	back, handler := openLoopback(t, api)

	right := back.Address(api.state.GetCurrentVault(), statOf(t, api, api.state.GetCurrentVault(), talk))
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
	api, _ := openRecordingWindow(t, nil)
	target := api.state.GetCurrentVault()
	find := vaults.NewFind(heldVaults{target})
	api.server.find = &find
	back, handler := openLoopback(t, api)

	// The window moves to a vault holding nothing of the kind.
	api.state.setVault(testsupport.NewVault(t, map[string]string{"other.md": "somewhere else"}))

	if out := ask(handler, back.Address(target, statOf(t, api, target, talk))); out.Code != http.StatusOK {
		t.Errorf("the recording of the vault that moved out of the window was answered %d: %s",
			out.Code, out.Body)
	}
}

// A vault this installation does not hold is not a vault to read from.
func TestAFileOfAVaultNobodyContainsIsRefused(t *testing.T) {
	api, _ := openRecordingWindow(t, nil)
	back, handler := openLoopback(t, api)

	elsewhere := api.state.GetCurrentVault()
	ref := statOf(t, api, elsewhere, talk)
	elsewhere.ID = "01ANOTHERVAULTALTOGETHER00"
	if out := ask(handler, back.Address(elsewhere, ref)); out.Code != http.StatusNotFound {
		t.Errorf("a vault nobody holds was answered %d", out.Code)
	}
}

// vault's own reader.
func TestTheSocketServesTheVaultAndNotTheDisk(t *testing.T) {
	vault := testsupport.NewVault(t, map[string]string{
		talk:      sound,
		"note.md": "# What a note says",
	})
	api := newBareServices()
	api.readFiles(vaultreader.Readers{})
	api.state.setVault(vault)
	back, handler := openLoopback(t, api)

	if out := ask(handler, back.Address(vault, statOf(t, api, vault, "note.md"))); out.Code != http.StatusOK {
		t.Errorf("a note of the vault was answered %d", out.Code)
	}
	for _, outside := range []string{"../secrets", "/etc/passwd"} {
		pos := back.Address(vault, statOf(t, api, vault, outside))
		if out := ask(handler, pos); out.Code == http.StatusOK {
			t.Errorf("%s was served", outside)
		}
	}
}

// A player asks for a piece and is answered with that piece.
func TestTheSocketAnswersAPiece(t *testing.T) {
	api, _ := openRecordingWindow(t, nil)
	back, handler := openLoopback(t, api)

	out := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, back.Address(api.state.GetCurrentVault(), statOf(t, api, api.state.GetCurrentVault(), talk)), nil)
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
	api, _ := openRecordingWindow(t, nil)
	back, err := Listen(api.server, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer back.Close()

	out, err := http.Get(back.Address(api.state.GetCurrentVault(), statOf(t, api, api.state.GetCurrentVault(), talk)))
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
	response, err := io.ReadAll(out.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(response) != sound {
		t.Errorf("it came back as %q", response)
	}
}

// socket a recording is played from is the one thing named beside it.
func TestThePolicyNamesTheSocketAndNothingElse(t *testing.T) {
	api, _ := openRecordingWindow(t, nil)
	back, err := Listen(api.server, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer back.Close()
	api.Playing = back

	response := getResponse(api.NewHandler(http.NotFoundHandler()), "/").
		Header().Get("Content-Security-Policy")
	if !strings.Contains(response, "media-src 'self' "+back.address) {
		t.Errorf("the socket is not what a recording may be played from: %q", response)
	}
	if strings.Contains(response, "img-src 'self' "+back.address) {
		t.Error("naming where a recording plays from widened where a picture comes from")
	}
}

const pointsAtAVideo = "https://youtu.be/dQw4w9WgXcQ"

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

func parseURL(t *testing.T, output string) domain.URL {
	t.Helper()
	pos, err := domain.ParseURL(output)
	if err != nil {
		t.Fatal(err)
	}
	return pos
}

func TestTheSchemaAndTheAddressCarryNanosecondsSinceTheEpoch(t *testing.T) {
	pos := time.Date(2026, 9, 3, 11, 4, 5, 123456789, time.UTC)
	const nanos = 1788433445123456789

	ref := domain.Fingerprint{Path: "library/Compost.pdf", Size: 12, ModTime: pos}
	if target := convertFingerprintToWire(ref).GetMtime(); target != nanos {
		t.Errorf("the schema carries mtime %d, want %d", target, nanos)
	}

	back := &Loopback{address: "http://127.0.0.1:1", token: "given-out"}
	want := "?size=12&mtime=1788433445123456789"
	if targetVal := back.Address(domain.Vault{ID: "01VAULT"}, ref); !strings.HasSuffix(targetVal, want) {
		t.Errorf("the address is %q, and it should end %q", targetVal, want)
	}
}

func TestAFileStampedByAClockIsStillItselfThroughTheSchema(t *testing.T) {
	ref := domain.Fingerprint{Path: "recordings/Compost.mp3", Size: 3, ModTime: time.Now()}
	if back := convertFingerprintFromWire(convertFingerprintToWire(ref)); !back.IsUnchanged(ref) {
		t.Errorf("the message carried the file back as %+v, and it is %+v", back, ref)
	}
}

func TestTheWindowsFingerprintIsAKeyOfPlainNumbers(t *testing.T) {
	target := reflect.TypeOf(document.Fingerprint{})
	if !target.Comparable() {
		t.Fatal("the window's fingerprint cannot be a map key at all")
	}
	for item := range target.NumField() {
		if field := target.Field(item); field.Type == reflect.TypeOf(time.Time{}) {
			t.Errorf("%s is a time.Time, and this struct is a map key", field.Name)
		}
	}
}

func TestAnEmptyStampIsNoStampAndNotTheEpoch(t *testing.T) {
	if back := convertFingerprintFromWire(convertFingerprintToWire(domain.Fingerprint{})); !back.IsZero() {
		t.Errorf("nothing said about a file came back as %+v", back)
	}
	if target := stamp(time.Time{}); target != 0 {
		t.Errorf("no stamp goes on the wire as %d", target)
	}
}
