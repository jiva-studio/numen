package editor

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/core/usecase/source"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// fileServer answers with one file of one vault, as it is read.
type fileServer struct {
	state    *State
	find     *vaults.Find
	openCopy *file.OpenCopy
	openFile *file.OpenFile
}

// A Loopback is a socket on the loopback address that answers with the vault's
// own files.
//
// The window is drawn from a scheme the toolkit registers, and a page fetched
// through it reaches this application. A player does not: a browser loads sound
// not the scheme of one application, and answers that it does not know the
// format. A loopback address is a place it does reach.
//
// Nothing about a recording is here. What the vault holds is what is served,
// and what may be played is decided where a player is drawn.
type Loopback struct {
	server  *http.Server
	files   *fileServer
	address string
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
func Listen(files *fileServer, stopped func(error)) (*Loopback, error) {
	target, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	back, err := newLoopback(target, files, stopped)
	if err != nil {
		target.Close()
		return nil, err
	}
	return back, nil
}

// newLoopback is the socket answering, whichever socket it is.
func newLoopback(target net.Listener, files *fileServer, stopped func(error)) (*Loopback, error) {
	word := make([]byte, 24)
	if _, err := rand.Read(word); err != nil {
		return nil, err
	}

	back := &Loopback{
		files:   files,
		address: "http://" + target.Addr().String(),
		token:   base64.RawURLEncoding.EncodeToString(word),
	}
	back.server = &http.Server{
		Handler:           back.getHandler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		err := back.server.Serve(target)
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

// getHandler answers for one file of one vault.
//
// The vault is named in the address and not taken from the window: a tab plays
// on while the person moves the window to another vault, and what it plays is
// the file it opened.
//
// the vault's own reader. That is the boundary, and it is the same one every
// other question crosses.
func (l *Loopback) getHandler() http.Handler {
	target := "/" + l.token + "/"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		escaped := r.URL.EscapedPath()
		if !strings.HasPrefix(escaped, target) {
			http.Error(w, "not an address this run gave out", http.StatusForbidden)
			return
		}
		id, rest, found := strings.Cut(strings.TrimPrefix(escaped, target), "/")
		if !found {
			http.Error(w, "not a file of a vault", http.StatusBadRequest)
			return
		}
		path, err := url.PathUnescape(rest)
		if err != nil {
			http.Error(w, "not a file of a vault", http.StatusBadRequest)
			return
		}
		l.files.File(w, r, id, path)
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
		"?" + formatFingerprint(ref)
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
func (h *fileServer) File(w http.ResponseWriter, r *http.Request, id, targetPath string) {
	named, err := parseFingerprint(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	target, ok := h.findVault(id)
	if !ok {
		http.Error(w, errNoVaultNamed, http.StatusNotFound)
		return
	}
	if h.openFile == nil {
		refuse(w, errNoVault)
		return
	}
	// A file a plugin keeps a copy of is played from the copy. Every other file of the vault is
	// served as the file it is.
	if h.serveCopy(w, r, target, targetPath, named) {
		return
	}
	opened, err := h.openFile.Execute(r.Context(), target, targetPath, named)
	if err != nil {
		refuse(w, err)
		return
	}
	defer opened.Body.Close()

	w.Header().Set("Cache-Control", immutable)
	if named := domain.MediaType(opened.Path); named != "" {
		w.Header().Set("Content-Type", named)
	}
	http.ServeContent(w, r, opened.Path, opened.ModTime, opened.Body)
}

// serveCopy answers with the copy a plugin keeps of the file, and says whether it answered at all.
// A file with no copy is served as the file it is.
//
// The size the address carries is the copy's own: a copy made again is a different address.
func (h *fileServer) serveCopy(
	w http.ResponseWriter, r *http.Request, target domain.Vault, path string, named domain.Fingerprint,
) bool {
	if h.openCopy == nil {
		return false
	}
	played, found, err := h.openCopy.Execute(r.Context(), target, path)
	if err != nil {
		refuse(w, err)
		return true
	}
	if !found {
		return false
	}
	defer played.Body.Close()
	if named.Size != played.Size {
		refuse(w, file.ErrChanged)
		return true
	}
	w.Header().Set("Cache-Control", immutable)
	w.Header().Set("Content-Type", played.MediaType)
	http.ServeContent(w, r, "", time.Time{}, played.Body)
	return true
}

// findVault is the vault an address names. The one the window shows is answered
// without asking the list, which is every question but the first.
func (h *fileServer) findVault(id string) (domain.Vault, bool) {
	if showing := h.state.GetCurrentVault(); string(showing.ID) == id {
		return showing, true
	}
	if h.find == nil {
		return domain.Vault{}, false
	}
	one, err := h.find.Execute(id)
	if err != nil {
		return domain.Vault{}, false
	}
	return one, true
}

// getPlayOrigins is where the window may play from, for the policy the page is served
// under. A build that opened no socket names nowhere.
func (l *Loopback) getPlayOrigins() []string {
	if l == nil {
		return nil
	}
	return []string{l.address}
}

const immutable = "public, max-age=31536000, immutable"

func formatFingerprint(ref domain.Fingerprint) string {
	return fmt.Sprintf("size=%d&mtime=%d", ref.Size, stamp(ref.ModTime))
}

func parseFingerprint(query url.Values) (domain.Fingerprint, error) {
	size, err := strconv.ParseInt(query.Get("size"), 10, 64)
	if err != nil {
		return domain.Fingerprint{}, fmt.Errorf("size: %q is not a size", query.Get("size"))
	}
	mtime, err := strconv.ParseInt(query.Get("mtime"), 10, 64)
	if err != nil {
		return domain.Fingerprint{}, fmt.Errorf("mtime: %q is not a time", query.Get("mtime"))
	}
	return domain.Fingerprint{Size: size, ModTime: instant(mtime)}, nil
}

func stamp(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}

func instant(nanos int64) time.Time {
	if nanos == 0 {
		return time.Time{}
	}
	return time.Unix(0, nanos)
}

func refuse(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, file.ErrChanged), source.IsNoNote(err):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, source.ErrOutside):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, errNoVault):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
