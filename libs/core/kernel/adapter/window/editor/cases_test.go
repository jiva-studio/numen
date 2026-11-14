package editor_test

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

func openRegisteredWindow(cfg container.Config, ctx context.Context, named string, out io.Writer) (*container.OpenedWindow, *container.WindowUseCases, error) {
	installation, err := container.OpenWindow(cfg, ctx, named, out)
	if err != nil {
		return nil, nil, err
	}
	return installation, &installation.UseCases, nil
}
