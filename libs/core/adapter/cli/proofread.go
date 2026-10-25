package cli

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// proofreadCommand puts one file right with a model, by what the file is: a
// recording's transcript, or a document's recognition.
func proofreadCommand(ctx context.Context, out io.Writer, deps Deps, args []string) error {
	if len(args) != 2 {
		return errors.New("usage: numen-cli proofread <vault> <file>")
	}
	v, err := findVault(deps, args[0])
	if err != nil {
		return err
	}
	if domain.MediaType(args[1]) != "" {
		return proofreadTranscriptCommand(ctx, out, deps, v, args[1])
	}
	return proofreadRecognitionCommand(ctx, out, deps, v, args[1])
}
