// Package service is the backend of the todos plugin: it serves the tasks of the notes to the
// plugin's frontend.
package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-todos/backend/usecase/index"
)

func mount(s *server.Server, tasks *index.Index) error {
	return nil
}

// Mount registers the service on s.
func Mount(s *server.Server) error {
	return mount(s, nil)
}
