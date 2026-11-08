package service

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

func newProofreadCommand(proofreading *Proofreading) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		source := domain.TextVersionID(req.Arguments.GetString(command.ArgumentVersion))
		res, err := proofreading.Proofread(ctx, Request{Vault: "", Path: req.Path, Source: source, IsUserStarted: true})
		if err != nil {
			return command.Result{}, err
		}
		message := fmt.Sprintf("proofread %d lines of %s and corrected %d", res.Lines, req.Path, res.Corrected)
		if res.Uncorrected > 0 {
			message += fmt.Sprintf("; the model returned no reply for %d batches", res.Uncorrected)
		}
		return command.Result{Message: message, OpenedPath: ""}, nil
	}
}
