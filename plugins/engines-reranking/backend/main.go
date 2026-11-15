// Command reranking serves the reranking plugin as a process.
package main

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/service"
)

func main() {
	server.Run(service.Mount)
}
