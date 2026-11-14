// Package service is the backend of the templates plugin: it makes notes from the templates of a folder
// of the vault and serves that to the plugin's frontend.
package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/usecase/newnote"
)

func normalizeFolder(raw string) string { return raw }

func mount(_ *server.Server, _ newnote.Vault, _ newnote.Clock, _ newnote.IDs) error { return nil }

// Mount registers the service on s.
func Mount(_ *server.Server) error { return nil }
