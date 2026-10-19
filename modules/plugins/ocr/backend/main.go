// Command ocr serves the OCR plugin as a process.
package main

import (
	"fmt"
	"os"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/modules/plugins/ocr/backend/service"
)

func main() {
	s := server.New()
	if err := service.Mount(s); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
	if err := s.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
