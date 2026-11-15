// Command main serves the vault knowledge tools plugin as a process.
package main

import (
	"log/slog"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/service"
)

func main() {
	server.Run(func(s *server.Server) error {
		return service.Mount(s, func(err error) {
			slog.Warn("the query tool starts without the functions of the search language", "error", err)
		})
	})
}
