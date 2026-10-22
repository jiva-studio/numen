package editor

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// fileServer answers with one file of one vault, as it is read.
type fileServer struct {
	state    *State
	files    *artifactFiles
	find     *vaults.Find
	openCopy source.OpenCopy
	openFile *document.OpenFile
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
		// The page holding a video's player is served here and names no vault:
		// it is a page of this application's own, and what it frames is a host.
		if id == embedRoute {
			l.serveFrame(w, r, rest)
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
		"?" + formatFingerprint(document.Fingerprint{Size: ref.Size, Mtime: stamp(ref.ModTime)})
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
	// A copy of what a url points at is kept in the vault's own folder,
	// was written to. Every other file of the vault is
	// served as the file it is.
	if strings.HasSuffix(targetPath, domain.URLExtension) && h.serveCopy(w, r, target, targetPath, named) {
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

// serveCopy answers with the copy fetched for a url, and says whether it
// answered at all. One with no copy on this disk is a file like any other,
// and is served as one.
//
// The size the address carries is the copy's own: a copy fetched again under
// the same name is a different address.
func (h *fileServer) serveCopy(
	w http.ResponseWriter, r *http.Request, target domain.Vault, pos string, named document.Fingerprint,
) bool {
	points := h.files.getArtifactAddress(r.Context(), target, domain.Fingerprint{Path: pos, Kind: domain.KindURL})
	if getPlayerURL(points) == "" {
		return false
	}
	video, found := h.openCopy.Execute(r.Context(), target, points)
	if !found {
		return false
	}
	defer video.File.Close()
	if named.Size != video.Size {
		refuse(w, document.ErrChanged)
		return true
	}
	w.Header().Set("Cache-Control", immutable)
	w.Header().Set("Content-Type", domain.CopyType)
	http.ServeContent(w, r, "", time.Time{}, video.File)
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

func formatFingerprint(mark document.Fingerprint) string {
	return fmt.Sprintf("size=%d&mtime=%d", mark.Size, mark.Mtime)
}

func parseFingerprint(query url.Values) (document.Fingerprint, error) {
	size, err := strconv.ParseInt(query.Get("size"), 10, 64)
	if err != nil {
		return document.Fingerprint{}, fmt.Errorf("size: %q is not a size", query.Get("size"))
	}
	mtime, err := strconv.ParseInt(query.Get("mtime"), 10, 64)
	if err != nil {
		return document.Fingerprint{}, fmt.Errorf("mtime: %q is not a time", query.Get("mtime"))
	}
	return document.Fingerprint{Size: size, Mtime: mtime}, nil
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
	case errors.Is(err, document.ErrChanged), source.IsNoNote(err):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, source.ErrOutside):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, errNoVault):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// embedRoute is where the page holding a player is served from.
const embedRoute = "embed"

// Embed is where a frame plays what is at an address, served over this socket.
//
// A host checks which page is framing its player, and it is told by the address
// that page was loaded from. A window drawn from a scheme of its own has no
// such address to give — a browser sends none for a scheme that is not http —
// and a host given none refuses to play. The page holding the player is
// therefore served here, over the socket this run already opened, and what the
// host is told is an address on this machine.
//
// It carries the address the note points at. Which player that address is
// played by is the domain's to say, and a host learned about later is played
// here without this route learning anything.
func (l *Loopback) Embed(targetURL domain.URL) string {
	if l == nil || l.stopped.Load() || getPlayerURL(targetURL) == "" {
		return ""
	}
	return l.address + "/" + l.token + "/" + embedRoute + "/" + url.PathEscape(string(targetURL))
}

// player is the page that holds one player.
//
// It carries the player and nothing else: no vault, no note, no token beyond
// the one in its own address. What it does carry is the one message a person's
// choosing sends — the moment a stretch of speech was said — which it takes
// from the page framing it and hands to the player.
var player = template.Must(template.New("player").Parse(
	`<!doctype html><meta charset="utf-8"><title>{{.Title}}</title>
<style>html,body{margin:0;height:100%;background:#000}iframe{border:0;width:100%;height:100%}</style>
<iframe id="player" src="{{.Player}}" allow="fullscreen; picture-in-picture"
  sandbox="allow-scripts allow-same-origin allow-popups"></iframe>
<script>
// This page is a player and nothing else, so it offers nothing of a browser's.
document.addEventListener('contextmenu', function (asked) { asked.preventDefault() })
window.addEventListener('message', function (said) {
  if (said.source !== window.parent) return
  var frame = document.getElementById('player')
  if (frame && frame.contentWindow) frame.contentWindow.postMessage(said.data, {{.Host}})
})
</script>`))

// serveFrame serves that page for one address.
func (l *Loopback) serveFrame(w http.ResponseWriter, _ *http.Request, raw string) {
	content, err := url.PathUnescape(raw)
	if err != nil {
		http.Error(w, "not an address", http.StatusBadRequest)
		return
	}
	pos, err := domain.ParseURL(content)
	if err != nil {
		http.Error(w, "not an address", http.StatusBadRequest)
		return
	}
	played := getPlayerURL(pos)
	if played == "" {
		http.Error(w, "nothing plays what is at that address", http.StatusBadRequest)
		return
	}
	host, ok := getFrameOrigin(played)
	if !ok {
		http.Error(w, "nothing plays what is at that address", http.StatusBadRequest)
		return
	}
	// This page frames the one host that plays this address, runs the one script
	// written into it, and reaches nowhere else at all.
	w.Header().Set("Content-Security-Policy",
		"default-src 'none'; frame-src "+host+"; script-src 'unsafe-inline'; style-src 'unsafe-inline'")
	// A host checks who is framing its player and is told by the address this
	// page cannot take that address away: a page that hands a host nothing plays
	// nothing.
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = player.Execute(w, struct {
		Title  string
		Player template.URL
		Host   template.JSStr
	}{
		Title:  string(pos),
		Player: template.URL(played + "&origin=" + url.QueryEscape(l.address)),
		Host:   template.JSStr(host),
	})
}

// getFrameOrigin is the origin a player is played from, and whether the window may
// the list and what composes an address are one decision, and this is where the
func getFrameOrigin(played string) (string, bool) {
	address, err := url.Parse(played)
	if err != nil {
		return "", false
	}
	host := address.Scheme + "://" + address.Host
	return host, slices.Contains(embedHosts, host)
}

// embedded is where a frame plays a video from. The host serves no cookies of
// its own, and `enablejsapi` is what makes the frame answer the page holding
// it, so a passage is played from the second it was said without that host's
// script running inside the window.
const embedded = "https://www.youtube-nocookie.com/embed/"

// embedHosts are the origins this window may frame. Every one of them runs its
// own scripts inside its own frame and reaches its own machines.
var embedHosts = []string{"https://www.youtube-nocookie.com"}

// videoSites are the hosts whose players this window knows how to frame. A host
// is written without `www.`, which is trimmed before the lookup.
var videoSites = map[string]bool{
	"youtube.com":          true,
	"m.youtube.com":        true,
	"music.youtube.com":    true,
	"youtube-nocookie.com": true,
	"youtu.be":             true,
}

// getPlayerURL is where a frame plays what is at an address, and nothing where this
// window knows no player for it.
func getPlayerURL(pos domain.URL) string {
	address, err := url.Parse(string(pos))
	if err != nil || !videoSites[strings.TrimPrefix(address.Hostname(), "www.")] {
		return ""
	}
	video := videoAt(address)
	if video == "" {
		return ""
	}
	return embedded + video + "?enablejsapi=1"
}

// videoAt is the video an address names, and nothing where it names none.
func videoAt(address *url.URL) string {
	if strings.TrimPrefix(address.Hostname(), "www.") == "youtu.be" {
		return videoID(strings.TrimPrefix(address.Path, "/"))
	}
	if address.Path == "/watch" {
		return videoID(address.Query().Get("v"))
	}
	for _, page := range []string{"/embed/", "/shorts/", "/live/", "/v/"} {
		if rest, found := strings.CutPrefix(address.Path, page); found {
			return videoID(rest)
		}
	}
	return ""
}

// videoID is the identifier where the segment is one, and nothing otherwise. It
// is eleven characters of the alphabet a URL carries unescaped, and a segment
// carrying anything else is a page of the site.
func videoID(segment string) string {
	segment, _, _ = strings.Cut(segment, "/")
	if len(segment) != 11 {
		return ""
	}
	for _, r := range segment {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_':
		default:
			return ""
		}
	}
	return segment
}
