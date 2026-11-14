package link

import (
	"bytes"
	"context"
	"html/template"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/address"
)

// pageExt ends the name of the page that frames the player of a link's video.
const pageExt = ".html"

// playerPage is the page that holds one player and nothing else. The player is sent the origin of
// the page as the Referer, which the site checks before it plays.
const playerPage = `<!doctype html>
<html><head><meta charset="utf-8"><title>{{.Address}}</title>
<style>html,body{margin:0;height:100%;background:#000}iframe{border:0;width:100%;height:100%}</style>
</head><body>
<iframe src="{{.Player}}" title="{{.Address}}" allow="fullscreen; picture-in-picture; encrypted-media"
  referrerpolicy="strict-origin-when-cross-origin"
  sandbox="allow-scripts allow-same-origin allow-popups allow-presentation"></iframe>
</body></html>
`

func writePage(ctx context.Context, store host.StoreWriter, key string, link address.URL) (string, bool, error) {
	player, ok := address.GetPlayerURL(link)
	if !ok {
		return "", false, nil
	}
	parsed, err := template.New("player").Parse(playerPage)
	// gomutants:disable-next-line BRANCH_IF reason="the constant page always parses"
	if err != nil {
		return "", false, err
	}
	var page bytes.Buffer
	// gomutants:disable-next-line BRANCH_IF reason="the template writes to a buffer, which does not fail"
	if err := parsed.Execute(&page, struct {
		Address string
		Player  string
	}{Address: string(link), Player: player}); err != nil {
		return "", false, err
	}
	name := key + pageExt
	return name, true, store.Write(ctx, name, page.Bytes())
}
