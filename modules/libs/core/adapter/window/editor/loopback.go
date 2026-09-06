package editor

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	derived "github.com/jiva-studio/numen/modules/libs/core/text"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// A Loopback is a socket on the loopback address that answers with the vault's
// own files.
//
// The window is drawn from a scheme the toolkit registers, and a page fetched
// through it reaches this application. A player does not: a browser loads sound
// and video down a path of its own, which speaks the protocols of the world and
// not the scheme of one application, and answers that it does not know the
// format. A loopback address is a place it does reach.
//
// Nothing about a recording is here. What the vault holds is what is served,
// and what may be played is decided where a player is drawn.
type Loopback struct {
	server  *http.Server
	api     *API
	address string
	// token stands in the address and in no setting. Every process on this
	// machine can reach a loopback socket, and a vault is the person's own
	// writing; the address is what tells this window's own asking from anybody
	// else's. It is made afresh for each run and outlives none of them.
	token string
	// stopped is set where the socket stopped answering for any reason but this
	// window closing. An address on a socket that answers nothing is a recording
	// that will not play and nothing said about why.
	stopped atomic.Bool
}

// Listen opens the socket. A machine that refuses one leaves whatever cannot
// reach the window's own scheme with nothing.
//
// stopped is told where the socket stops answering under the window, which is
// nobody's doing and nothing anybody asked for.
func Listen(api *API, stopped func(error)) (*Loopback, error) {
	held, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	back, err := answering(held, api, stopped)
	if err != nil {
		held.Close()
		return nil, err
	}
	return back, nil
}

// answering is the socket answering, whichever socket it is.
func answering(held net.Listener, api *API, stopped func(error)) (*Loopback, error) {
	word := make([]byte, 24)
	if _, err := rand.Read(word); err != nil {
		return nil, err
	}

	back := &Loopback{
		api:     api,
		address: "http://" + held.Addr().String(),
		token:   base64.RawURLEncoding.EncodeToString(word),
	}
	back.server = &http.Server{
		Handler:           back.serving(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		err := back.server.Serve(held)
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		back.stopped.Store(true)
		if stopped != nil {
			stopped(err)
		}
	}()
	return back, nil
}

// serving answers for one file of one vault.
//
// The vault is named in the address and not taken from the window: a tab plays
// on while the person moves the window to another vault, and what it plays is
// the file it opened.
//
// A path leaving the vault, and a file the vault leaves alone, are refused by
// the vault's own reader. That is the boundary, and it is the same one every
// other question crosses.
func (l *Loopback) serving() http.Handler {
	held := "/" + l.token + "/"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		escaped := r.URL.EscapedPath()
		if !strings.HasPrefix(escaped, held) {
			http.Error(w, "not an address this run gave out", http.StatusForbidden)
			return
		}
		id, rest, found := strings.Cut(strings.TrimPrefix(escaped, held), "/")
		if !found {
			http.Error(w, "not a file of a vault", http.StatusBadRequest)
			return
		}
		path, err := url.PathUnescape(rest)
		if err != nil {
			http.Error(w, "not a file of a vault", http.StatusBadRequest)
			return
		}
		l.api.File(w, r, id, path)
	})
}

// Address is where the file at a path in a vault is read from. A build that
// opened no socket, and a socket that stopped answering, both answer with
// nowhere.
//
// The address names which bytes the file was when it was given out, so a player
// loaded from it plays one recording through and is not spliced with another
// halfway. A file rewritten under the same name is a different address.
func (l *Loopback) Address(vault domain.Vault, ref domain.Fingerprint) string {
	if l == nil || l.stopped.Load() {
		return ""
	}
	return l.address + "/" + l.token + "/" + url.PathEscape(string(vault.ID)) +
		"/" + url.PathEscape(ref.Path) +
		"?" + printing(fingerprint{size: ref.Size, mtime: stamp(ref.ModTime)})
}

// Close stops answering.
func (l *Loopback) Close() error {
	if l == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return l.server.Shutdown(ctx)
}

// errNoVaultNamed is what an address naming a vault this installation does not
// hold is answered with.
const errNoVaultNamed = "no vault of that name"

// File serves one file of one vault, as it is read.
//
// A range is answered as a range, so a player seeks in an hour of speech and
// holds the second it is on. The whole file is never in memory.
func (a *API) File(w http.ResponseWriter, r *http.Request, id, at string) {
	named, err := printed(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	held, ok := a.vaultOf(id)
	if !ok {
		http.Error(w, errNoVaultNamed, http.StatusNotFound)
		return
	}
	if a.Readers == nil {
		refuse(w, errNoVault)
		return
	}
	// A copy of what a link note points at is kept in the vault's own folder,
	// which the vault's reader is refused, so it is served from the store it
	// was written to. Every other file of the vault, this note included, is
	// served as the file it is.
	if strings.HasSuffix(at, domain.NoteExtension) && a.served(w, r, held, at, named) {
		return
	}
	reader, err := a.Readers.Open(held)
	if err != nil {
		refuse(w, err)
		return
	}
	ref, err := reader.Stat(r.Context(), at)
	if err != nil {
		refuse(w, err)
		return
	}
	if named.size != ref.Size || named.mtime != stamp(ref.ModTime) {
		refuse(w, errChanged)
		return
	}
	file, err := reader.Open(r.Context(), ref.Path)
	if err != nil {
		refuse(w, err)
		return
	}
	defer file.Close()

	w.Header().Set("Cache-Control", immutable)
	if named := domain.MediaType(ref.Path); named != "" {
		w.Header().Set("Content-Type", named)
	}
	http.ServeContent(w, r, ref.Path, ref.ModTime, file)
}

// served answers with the copy fetched for a link note, and says whether it
// answered at all. A note with no copy on this disk is a file like any other,
// and is served as one.
//
// The size the address carries is the copy's own: a copy fetched again under
// the same name is a different address.
func (a *API) served(
	w http.ResponseWriter, r *http.Request, held domain.Vault, at string, named fingerprint,
) bool {
	_, stores, ready := a.hearing()
	if !ready {
		return false
	}
	points := a.points(r.Context(), held, domain.Fingerprint{Path: at, Kind: domain.KindNote})
	if !points.IsVideo() {
		return false
	}
	store, err := stores.Open(held)
	if err != nil {
		return false
	}
	name := derived.Copy(derived.Fingerprint([]byte(points.URL)))
	file, size, err := store.Open(r.Context(), name)
	if err != nil {
		return false
	}
	defer file.Close()
	if named.size != size {
		refuse(w, errChanged)
		return true
	}
	w.Header().Set("Cache-Control", immutable)
	w.Header().Set("Content-Type", derived.CopyType)
	http.ServeContent(w, r, name, time.Time{}, file)
	return true
}

// vaultOf is the vault an address names. The one the window shows is answered
// without asking the list, which is every question but the first.
func (a *API) vaultOf(id string) (domain.Vault, bool) {
	if showing := a.Showing(); string(showing.ID) == id {
		return showing, true
	}
	if a.Vaults.Registry == nil {
		return domain.Vault{}, false
	}
	one, err := vaults.NewFind(a.Vaults.Registry).Execute(id)
	if err != nil {
		return domain.Vault{}, false
	}
	return one, true
}

// named is where the window may play from, for the policy the page is served
// under. A build that opened no socket names nowhere.
func (l *Loopback) named() []string {
	if l == nil {
		return nil
	}
	return []string{l.address}
}
