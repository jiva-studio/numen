package editor_test

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

func openRegistered(cfg container.Config, ctx context.Context, named string, out io.Writer) (*editor.Installation, *container.WindowUseCases, error) {
	cases := &container.WindowUseCases{}
	db, err := container.OpenIndex(cfg, ctx)
	if err != nil {
		return nil, nil, err
	}
	installation, err := container.OpenWindowWith(cfg, ctx, container.WindowOpening{DB: db, Named: named, Out: out, UseCases: cases})
	if err != nil {
		return nil, nil, err
	}
	return installation, cases, nil
}
