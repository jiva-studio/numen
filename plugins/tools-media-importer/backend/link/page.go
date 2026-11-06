package link

import (
	"bytes"
	"context"
	"html/template"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/address"
)

// pageExt ends the name of the page that frames the player of a link's video.
const pageExt = ".html"

// playerPage is the page that holds one player and nothing else. The player is sent the origin of
// the page as the Referer, which the site checks before it plays.
var playerPage = template.Must(template.New("player").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><title>{{.Address}}</title>
<style>html,body{margin:0;height:100%;background:#000}iframe{border:0;width:100%;height:100%}</style>
</head><body>
<iframe src="{{.Player}}" title="{{.Address}}" allow="fullscreen; picture-in-picture; encrypted-media"
  referrerpolicy="strict-origin-when-cross-origin"
  sandbox="allow-scripts allow-same-origin allow-popups allow-presentation"></iframe>
</body></html>
`))

// writePage writes the page that frames the player of the video at the address into the folder under
// the key, and returns its name; false where no player is known for the address.
func writePage(ctx context.Context, store host.Store, key string, link address.URL) (string, bool, error) {
	player, ok := address.GetPlayerURL(link)
	if !ok {
		return "", false, nil
	}
	var page bytes.Buffer
	// gomutants:disable-next-line BRANCH_IF reason="the template writes to a buffer, which does not fail"
	if err := playerPage.Execute(&page, struct {
		Address string
		Player  template.URL
	}{Address: string(link), Player: template.URL(player)}); err != nil {
		return "", false, err
	}
	name := key + pageExt
	return name, true, store.Write(ctx, name, page.Bytes())
}
