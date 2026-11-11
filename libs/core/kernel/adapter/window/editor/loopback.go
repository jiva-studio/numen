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

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
)

// FileServer answers with one file of one vault, as it is read.
type FileServer struct {
	vaults shownVault
	tasks  taskList
	open   *file.OpenVaultFile
}

// shownVault is the vault the window shows.
type shownVault interface {
	GetCurrentVault() domain.Vault
}

// taskList is where the work of the window is listed.
type taskList interface {
	AddTask(pos work.Task)
}

// Loopback is a socket on the loopback address that serves the vault's files.
// The window loads pages from a scheme registered with the toolkit, which the
// browser media stack does not accept; a loopback address it can reach.
// The Loopback holds no recording logic: what may be played is decided where
// the player is drawn.
type Loopback struct {
	server  *http.Server
	files   *FileServer
	address string
	token   string
	stopped atomic.Bool
	plugins atomic.Pointer[PluginServices]
}

// Listen opens the socket. When the machine rejects a listener, content that
// cannot use the window's own scheme is unavailable.
//
// stopped is called when the socket stops serving while the window is open.
func Listen(ctx context.Context, files *FileServer, stopped func(error)) (*Loopback, error) {
	target, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	back, err := newLoopback(target, files, stopped)
	if err != nil {
		return nil, errors.Join(err, target.Close())
	}
	return back, nil
}

func newLoopback(target net.Listener, files *FileServer, stopped func(error)) (*Loopback, error) {
	word := make([]byte, 24)
	if _, err := rand.Read(word); err != nil {
		return nil, err
	}

	back := &Loopback{
		files:   files,
		address: (&url.URL{Scheme: "http", Host: target.Addr().String()}).String(),
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
		if id == pluginFileSegment {
			l.servePluginFile(w, r, rest)
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

// Address returns where the file at a path in a vault is read from, or nowhere
// for a build that opened no socket or a socket that stopped answering.
// The address names the bytes the file had when it was given out, so a player
// never splices two recordings. A file rewritten under the same name is a
// different address.
func (l *Loopback) Address(vault domain.Vault, ref domain.FileStat) string {
	if l == nil || l.stopped.Load() {
		return ""
	}
	return l.address + "/" + l.token + "/" + url.PathEscape(string(vault.ID)) +
		"/" + url.PathEscape(ref.Path) +
		"?" + formatFingerprint(ref)
}

// Close stops answering. ctx bounds the wait for the answers under way.
func (l *Loopback) Close(ctx context.Context) error {
	if l == nil {
		return nil
	}
	return l.server.Shutdown(ctx)
}

// errNoVaultNamed is what an address naming a vault this installation does not
// hold is answered with.
const errNoVaultNamed = "no vault of that name"

// File serves one file of one vault, as it is read.
//
// A range is answered as a range, so a player seeks in an hour of speech and
// holds the second it is on. The whole file is never in memory.
func (h *FileServer) File(w http.ResponseWriter, r *http.Request, id, targetPath string) {
	named, err := parseFingerprint(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if h.open == nil {
		refuse(w, errNoVault)
		return
	}
	opened, err := h.open.Execute(r.Context(), h.vaults.GetCurrentVault(), id, targetPath, named)
	if errors.Is(err, file.ErrNoVaultNamed) {
		http.Error(w, errNoVaultNamed, http.StatusNotFound)
		return
	}
	if err != nil {
		refuse(w, err)
		return
	}
	defer closeAndReport(h.tasks.AddTask, opened.Body)

	w.Header().Set("Cache-Control", immutable)
	if opened.MediaType != "" {
		w.Header().Set("Content-Type", opened.MediaType)
	}
	http.ServeContent(w, r, opened.Path, opened.ModTime, opened.Body)
}

func (l *Loopback) getPlayOrigins() []string {
	if l == nil {
		return nil
	}
	return []string{l.address}
}

const immutable = "public, max-age=31536000, immutable"

func formatFingerprint(ref domain.FileStat) string {
	return fmt.Sprintf("size=%d&mtime=%d", ref.Size, stamp(ref.ModTime))
}

func parseFingerprint(query url.Values) (domain.FileStat, error) {
	size, err := strconv.ParseInt(query.Get("size"), 10, 64)
	if err != nil {
		return domain.FileStat{}, fmt.Errorf("size: %q is not a size", query.Get("size"))
	}
	mtime, err := strconv.ParseInt(query.Get("mtime"), 10, 64)
	if err != nil {
		return domain.FileStat{}, fmt.Errorf("mtime: %q is not a time", query.Get("mtime"))
	}
	return domain.FileStat{Size: size, ModTime: instant(mtime)}, nil
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
	case errors.Is(err, file.ErrChanged), edit.IsNoNote(err):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, vault.ErrOutside):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, errNoVault):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
