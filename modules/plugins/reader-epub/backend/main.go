package main

import (
	"fmt"
	"os"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/modules/plugins/reader-epub/backend/service"
)

func main() {
	s := server.New()
	book := service.NewBookServer()

	s.Register(numenv1connect.NewBookServiceHandler(book, s.HandlerOptions()...))

	if err := s.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
