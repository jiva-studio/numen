// Package main runs the Antigravity agent plugin backend.
package main

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/service"
)

func main() {
	server.Run(service.Mount)
}
