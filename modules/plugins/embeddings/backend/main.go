// Command embeddings serves the embeddings plugin as a process.
package main

import (
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/modules/plugins/embeddings/backend/service"
)

func main() {
	server.Run(service.Mount)
}

