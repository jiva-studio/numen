package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/process"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-updater/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-updater/backend/internal/adapter/feed"
	"github.com/jiva-studio/numen/plugins/tools-updater/backend/usecase/updates"
)

func mount(s *server.Server, fetcher updates.Fetcher, releaser Releaser) error {
	checker := updates.NewChecker(fetcher)
	s.SetUpdateConfig(func(values map[string]string) { checker.Configure(values) })
	a := s.NewAPI(api.Service)
	server.Handle(a, api.MethodCheck, newCheckHandler(checker, releaser))
	server.Handle(a, api.MethodGetStatus, newStatusHandler(checker))
	// gomutants:disable-next-line RETURN_ERROR_NIL reason="Service and method names are valid compile-time constants."
	return a.Err()
}

// Mount registers the updater API on s. The manifest is requested over the network, and each check
// releases the process through the host the environment names.
func Mount(s *server.Server) error {
	var releaser Releaser
	if client, err := process.NewClientFromEnv(); err == nil {
		releaser = client
	}
	return mount(s, feed.New(), releaser)
}
