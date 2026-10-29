// Package main runs the ACP agent plugin backend.
package main

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/service"
)

func main() {
	server.Run(service.Mount)
}
