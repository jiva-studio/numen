package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-updater/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-updater/backend/usecase/updates"
)

// Releaser tells the host that the process has finished its work.
type Releaser interface {
	Release(ctx context.Context) error
}

func newCheckHandler(checker *updates.Checker, releaser Releaser) func(context.Context, api.CheckRequest) (api.Status, error) {
	return func(ctx context.Context, req api.CheckRequest) (api.Status, error) {
		defer releaseProcess(ctx, releaser)
		app := updates.App{Version: req.App.Version, Build: req.App.Build, Platform: req.App.Platform}
		status, err := checker.Check(ctx, app)
		if err != nil {
			return api.Status{}, getAPIError(err)
		}
		return writeStatus(status), nil
	}
}

func releaseProcess(ctx context.Context, releaser Releaser) {
	if releaser == nil {
		return
	}
	if err := releaser.Release(ctx); err != nil {
		slog.Warn("updater: release the process", "error", err)
	}
}

func getAPIError(err error) error {
	var platform *updates.PlatformError
	switch {
	case errors.Is(err, updates.ErrNotConfigured):
		return server.NewAPIError(server.CodeFailedPrecondition, err.Error())
	case errors.As(err, &platform), errors.Is(err, updates.ErrUnknownVersion):
		return server.NewAPIError(server.CodeInvalidArgument, err.Error())
	default:
		return err
	}
}
