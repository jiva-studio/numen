package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-updater/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-updater/backend/feed"
	"github.com/jiva-studio/numen/plugins/tools-updater/backend/updates"
)

func mount(s *server.Server, fetcher updates.Fetcher) error {
	checker := updates.NewChecker(fetcher)
	s.SetUpdateConfig(func(values map[string]string, _ server.Secrets) { checker.Configure(values) })
	a := s.NewAPI(api.Service)
	server.Handle(a, api.MethodCheck, newCheckHandler(checker))
	server.Handle(a, api.MethodGetStatus, newStatusHandler(checker))
	// gomutants:disable-next-line RETURN_ERROR_NIL reason="Service and method names are valid compile-time constants."
	return a.Err()
}

// Mount registers the updater API on s. The manifest is requested over the network.
func Mount(s *server.Server) error {
	return mount(s, feed.New())
}
