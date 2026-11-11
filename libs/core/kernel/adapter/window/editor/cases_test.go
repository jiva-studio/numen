package editor_test

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

func openRegistered(cfg container.Config, ctx context.Context, named string, out io.Writer) (*container.OpenedWindow, *container.WindowUseCases, error) {
	db, err := container.OpenIndex(cfg, ctx)
	if err != nil {
		return nil, nil, err
	}
	return openOver(cfg, ctx, db, named, out)
}

func openOver(cfg container.Config, ctx context.Context, db *container.Index, named string, out io.Writer) (*container.OpenedWindow, *container.WindowUseCases, error) {
	cases := &container.WindowUseCases{}
	installation, err := container.OpenWindowWith(cfg, ctx, container.WindowOpening{DB: db, Named: named, Out: out, UseCases: cases})
	if err != nil {
		return nil, nil, err
	}
	return installation, cases, nil
}
