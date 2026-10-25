// Command downloader serves the media downloader plugin as a process.
package main

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/service"
)

func main() {
	server.Run(service.Mount)
}
