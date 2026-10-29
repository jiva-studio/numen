package service

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// newProofreadCommand proofreads the version the host names, or the live text of the file where it
// names none, and waits for it.
func newProofreadCommand(proofreading *Proofreading) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		source := host.TextVersionID(req.Arguments.GetString(command.ArgumentVersion))
		res, err := proofreading.Proofread(ctx, Request{Path: req.Path, Source: source, IsUserStarted: true})
		if err != nil {
			return command.Result{}, err
		}
		message := fmt.Sprintf("proofread %d lines of %s and put %d right", res.Lines, req.Path, res.Corrected)
		if res.Uncorrected > 0 {
			message += fmt.Sprintf("; the model gave no answer for %d batches", res.Uncorrected)
		}
		return command.Result{Message: message}, nil
	}
}
