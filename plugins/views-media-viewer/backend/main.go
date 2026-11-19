// Command main serves the audio recorder.
package main

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-media-viewer/backend/service"
)

func main() {
	server.Run(service.Mount)
}
