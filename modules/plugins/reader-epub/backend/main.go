package main

import (
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/modules/plugins/reader-epub/backend/service"
)

func main() {
	server.Run(service.Mount)
}

