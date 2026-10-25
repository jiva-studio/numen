// Command ocr serves the OCR plugin as a process.
package main

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/service"
)

func main() {
	server.Run(service.Mount)
}
