package main

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/service"
)

func main() {
	server.Run(service.Mount)
}
