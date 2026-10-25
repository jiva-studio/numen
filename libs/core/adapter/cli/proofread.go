package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// proofreadCommand puts a recording's text right through the plugin that reads it.
func proofreadCommand(ctx context.Context, out io.Writer, deps Deps, args []string) error {
	if len(args) != 2 {
		return errors.New("usage: numen-cli proofread <vault> <file>")
	}
	v, err := findVault(deps, args[0])
	if err != nil {
		return err
	}
	if domain.MediaType(args[1]) == "" {
		return fmt.Errorf("%s is no recording: numen-cli proofread puts a recording's text right", args[1])
	}
	return proofreadTextCommand(ctx, out, deps, v, args[1])
}
