package command

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// NewDeleteTextRun is the run of a command that deletes a text version the plugin of the id made. The
// version is the ArgumentVersion the host gives the command; the file is the one the call names.
func NewDeleteTextRun(texts host.Texts, store host.Store, pluginID string) Run {
	return func(ctx context.Context, request Request) (Result, error) {
		id := request.Arguments.GetString(ArgumentVersion)
		if id == "" {
			return Result{}, fmt.Errorf("%w: %s needs %s", ErrBadArguments, request.Command, ArgumentVersion)
		}
		if err := host.DeleteText(ctx, texts, store, pluginID, request.Path, host.TextVersionID(id)); err != nil {
			return Result{}, err
		}
		return Result{Message: "deleted " + id + " of " + request.Path}, nil
	}
}
