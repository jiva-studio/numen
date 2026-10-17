// Command main is the monolithic entry point: desktop application with plugins wired in-process.
package main

import (
	"fmt"
	"os"

	"github.com/jiva-studio/numen/modules/apps/desktop/app"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	epubservice "github.com/jiva-studio/numen/modules/plugins/reader-epub/backend/service"
	pdfservice "github.com/jiva-studio/numen/modules/plugins/reader-pdf/backend/service"
)

func main() {
	wireMonolithHandlers(appRegistry{})
	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "numen:", err)
		os.Exit(1)
	}
}

// handlerRegistry is where the monolith registers its in-process handlers.
type handlerRegistry interface {
	RegisterDocumentHandler(numenv1connect.DocumentServiceHandler)
	RegisterBookHandler(numenv1connect.BookServiceHandler)
}

type appRegistry struct{}

func (appRegistry) RegisterDocumentHandler(h numenv1connect.DocumentServiceHandler) {
	app.RegisterDocumentHandler(h)
}

func (appRegistry) RegisterBookHandler(h numenv1connect.BookServiceHandler) {
	app.RegisterBookHandler(h)
}

// wireMonolithHandlers registers the handlers that do real work. A capability
// with no handler stays unregistered, so the host reports its plugin as
// unavailable.
func wireMonolithHandlers(r handlerRegistry) {
	r.RegisterDocumentHandler(pdfservice.NewDocumentServer())
	r.RegisterBookHandler(epubservice.NewBookServer())
}
