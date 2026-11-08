package service

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/address"
)

// embedPattern is where the page that frames a video's player is served below the plugin route.
// The window requests /plugin/md.numen.tools.media.importer/embed?address=<web address>.
const embedPattern = "embed"

func getEmbedPage(link address.URL) string {
	if _, ok := address.GetPlayerURL(link); !ok {
		return ""
	}
	return embedPattern + "?" + url.Values{"address": {string(link)}}.Encode()
}

// embedPage is the page that holds one player and nothing else. The player is sent the origin
// of the page as the Referer, which the site checks before it plays.
const embedPage = `<!doctype html>
<html><head><meta charset="utf-8"><title>{{.Address}}</title>
<style>html,body{margin:0;height:100%;background:#000}iframe{border:0;width:100%;height:100%}</style>
</head><body>
<iframe src="{{.Player}}" title="{{.Address}}" allow="fullscreen; picture-in-picture; encrypted-media"
  referrerpolicy="strict-origin-when-cross-origin"
  sandbox="allow-scripts allow-same-origin allow-popups allow-presentation"></iframe>
</body></html>
`

func serveEmbed(w http.ResponseWriter, r *http.Request) {
	parsed, err := address.Parse(r.URL.Query().Get("address"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	player, ok := address.GetPlayerURL(parsed)
	if !ok {
		http.Error(w, "no player is known for this address", http.StatusNotFound)
		return
	}
	page, err := template.New("embed").Parse(embedPage)
	// gomutants:disable-next-line BRANCH_IF reason="constant embedPage template always parses"
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// gomutants:disable-next-line STATEMENT_REMOVE reason="template execution error only logs to stderr"
	err = page.Execute(w, struct {
		Address string
		Player  template.URL
	}{Address: string(parsed), Player: template.URL(player)})
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="stderr diagnostic log on response write failure has no observable effect"
	if err != nil {
		fmt.Fprintf(os.Stderr, "tools-media-importer: write the embed page: %v\n", err)
	}
}
