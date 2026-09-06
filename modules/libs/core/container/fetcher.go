package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/fetch"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Fetcher is what reaches an address on this machine, and nothing where the
// machine holds neither of the tools that reach one.
//
// A build with no fetcher is a build that cannot import: the run is answered
// that it cannot be done here, and the window offers it nowhere from then on.
func (c Config) Fetcher(ctx context.Context) port.Fetcher {
	fetcher, err := fetch.New(ctx, c.Fetching)
	if err != nil {
		c.trouble(err)
		return nil
	}
	return fetcher
}
