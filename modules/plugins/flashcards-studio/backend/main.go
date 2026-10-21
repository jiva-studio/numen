// Command flashcards-studio serves the flashcards-studio plugin as a process.
package main

import (
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/service"
)

func main() {
	server.Run(service.Mount)
}

