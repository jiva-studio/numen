package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/process"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/usecase/calibration"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/usecase/relevance"
)

// NewAppOver returns the backend over the hosts given, with measurements stamped by the clock, that
// releases its process through releaser. A nil releaser releases nothing.
func NewAppOver(hosts Hosts, clk relevance.Clock, releaser process.Releaser) *App {
	return &App{
		open:  func() (Hosts, error) { return hosts, nil },
		clock: clk, releaser: releaser, calibration: calibration.New(),
	}
}
