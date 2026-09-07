package editor

import (
	"html/template"
	"net/http"
	"net/url"
	"slices"

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
func (l *Loopback) Embed(at domain.WebAddress) string {
	if l == nil || l.stopped.Load() || at.Embed() == "" {
		return ""
	}
	return l.address + "/" + l.token + "/" + embedRoute + "/" + url.PathEscape(at.URL)
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

// framing serves that page for one address.
func (l *Loopback) framing(w http.ResponseWriter, r *http.Request, raw string) {
	written, err := url.PathUnescape(raw)
	if err != nil {
		http.Error(w, "not an address", http.StatusBadRequest)
		return
	}
	at, err := domain.ParseWebAddress(written)
	if err != nil {
		http.Error(w, "not an address", http.StatusBadRequest)
		return
	}
	played := at.Embed()
	if played == "" {
		http.Error(w, "nothing plays what is at that address", http.StatusBadRequest)
		return
	}
	host, ok := framed(played)
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
		Title:  at.URL,
		Player: template.URL(played + "&origin=" + url.QueryEscape(l.address)),
		Host:   template.JSStr(host),
	})
}

// framed is the origin a player is played from, and whether the window may
// frame it at all. A player composed for a host nobody named is played nowhere:
// the list and what composes an address are one decision, and this is where the
// two are held to each other.
func framed(played string) (string, bool) {
	address, err := url.Parse(played)
	if err != nil {
		return "", false
	}
	host := address.Scheme + "://" + address.Host
	return host, slices.Contains(domain.EmbedHosts(), host)
}
