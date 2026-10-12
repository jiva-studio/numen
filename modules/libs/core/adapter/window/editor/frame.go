package editor

import (
	"html/template"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

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
func (l *Loopback) Embed(at domain.URL) string {
	if l == nil || l.stopped.Load() || getPlayerURL(at) == "" {
		return ""
	}
	return l.address + "/" + l.token + "/" + embedRoute + "/" + url.PathEscape(string(at))
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
func (l *Loopback) serveFrame(w http.ResponseWriter, r *http.Request, raw string) {
	written, err := url.PathUnescape(raw)
	if err != nil {
		http.Error(w, "not an address", http.StatusBadRequest)
		return
	}
	at, err := domain.ParseURL(written)
	if err != nil {
		http.Error(w, "not an address", http.StatusBadRequest)
		return
	}
	played := getPlayerURL(at)
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
	// page stands at. The policy is written here so that whatever frames this
	// page cannot take that address away: a page that hands a host nothing plays
	// nothing.
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = player.Execute(w, struct {
		Title  string
		Player template.URL
		Host   template.JSStr
	}{
		Title:  string(at),
		Player: template.URL(played + "&origin=" + url.QueryEscape(l.address)),
		Host:   template.JSStr(host),
	})
}

// getFrameOrigin is the origin a player is played from, and whether the window may
// frame it at all. A player composed for a host nobody named is played nowhere:
// the list and what composes an address are one decision, and this is where the
// two are held to each other.
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
func getPlayerURL(at domain.URL) string {
	address, err := url.Parse(string(at))
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
