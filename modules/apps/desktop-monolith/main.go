// Command main is the monolithic entry point: desktop application with plugins wired in-process.
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/jiva-studio/numen/modules/apps/desktop/app"
	readerepub "github.com/jiva-studio/numen/modules/plugins/reader-epub"
	epubservice "github.com/jiva-studio/numen/modules/plugins/reader-epub/backend/service"
	readerpdf "github.com/jiva-studio/numen/modules/plugins/reader-pdf"
	pdfservice "github.com/jiva-studio/numen/modules/plugins/reader-pdf/backend/service"
)

func main() {
	registerPlugins(app.RegisterPlugin)
	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "numen:", err)
		os.Exit(1)
	}
}

// serveFunc makes the handler of a plugin's backend from the variables a launched process of the
// plugin is given.
type serveFunc = func(vars map[string]string) (http.Handler, error)

// registerPlugins registers each plugin this binary links in, with its own manifest and backend.
func registerPlugins(register func(manifest []byte, serve serveFunc)) {
	register(readerpdf.Manifest, pdfservice.NewHandler)
	register(readerepub.Manifest, epubservice.NewHandler)
}
