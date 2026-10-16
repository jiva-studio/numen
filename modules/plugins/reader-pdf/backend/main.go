package main

import (
	"fmt"
	"os"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/modules/plugins/reader-pdf/backend/service"
)

func main() {
	s := server.New()
	doc := service.NewDocumentServer()

	s.Register(numenv1connect.NewDocumentServiceHandler(doc, s.HandlerOptions()...))

	if err := s.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
