package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/ai"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// proofreaderName is the name the proofreader goes under.
const proofreaderName = "llm-proofreader"

// purpose is what every question to the model is for. The host routes by it.
const purpose = "proofreading"

// completer asks the host for the answer to a question.
type completer func(ctx context.Context, question *numenv1.CompletionQuestion) (string, error)

type proofreadingServer struct {
	ask askModel
}

// newProofreadingServer returns a server whose model is the one the host routes
// the proofreading purpose to.
func newProofreadingServer(complete completer) *proofreadingServer {
	return &proofreadingServer{ask: newAskModel(complete)}
}

// newAskModel returns the model as a system instruction and a prompt are put to it.
func newAskModel(complete completer) askModel {
	return func(ctx context.Context, instruction, prompt string) (string, error) {
		temperature := 0.0
		return complete(ctx, &numenv1.CompletionQuestion{
			Purpose: purpose,
			Messages: []*numenv1.CompletionMessage{
				{Role: "system", Text: instruction},
				{Role: "user", Text: prompt},
			},
			Hints: &numenv1.CompletionHints{Temperature: &temperature},
		})
	}
}

func (s *proofreadingServer) GetProofreaderInfo(
	_ context.Context,
	_ *connect.Request[numenv1.GetProofreaderInfoRequest],
) (*connect.Response[numenv1.GetProofreaderInfoResponse], error) {
	return connect.NewResponse(&numenv1.GetProofreaderInfoResponse{Name: proofreaderName}), nil
}

// Proofread answers each batch with the model's reply to its prompt.
func (s *proofreadingServer) Proofread(
	ctx context.Context,
	req *connect.Request[numenv1.ProofreadRequest],
) (*connect.Response[numenv1.ProofreadResponse], error) {
	batches := req.Msg.GetBatches()
	replies := make(map[int32]string, len(batches))
	for _, batch := range batches {
		reply, err := s.ask(ctx, req.Msg.GetInstruction(), batch.GetPrompt())
		if err != nil {
			return nil, readModelError(ctx, fmt.Errorf("batch %d: %w", batch.GetNumber(), err))
		}
		replies[batch.GetNumber()] = reply
	}
	return connect.NewResponse(&numenv1.ProofreadResponse{Replies: replies}), nil
}

// ProofreadText proofreads the lines of a request according to its kind.
func (s *proofreadingServer) ProofreadText(
	ctx context.Context,
	req *connect.Request[numenv1.ProofreadTextRequest],
) (*connect.Response[numenv1.ProofreadTextResponse], error) {
	result, err := s.proofread(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	corrections := make([]*numenv1.ProofreadCorrection, len(result.corrections))
	for i, line := range result.corrections {
		corrections[i] = &numenv1.ProofreadCorrection{
			Number: int32(line.Number),
			Last:   int32(line.Last),
			Text:   line.Text,
		}
	}
	return connect.NewResponse(&numenv1.ProofreadTextResponse{
		Corrections: corrections,
		Uncorrected: int32(result.uncorrected),
		NextLine:    int32(result.next),
		IsDone:      result.isDone,
	}), nil
}

// proofread runs a request under the kind it names. A kind with no options of
// its own, and options of the other kind, are refused.
func (s *proofreadingServer) proofread(ctx context.Context, msg *numenv1.ProofreadTextRequest) (textResult, error) {
	groups := readGroups(msg.GetGroups())
	var (
		result textResult
		err    error
	)
	switch msg.GetKind() {
	case numenv1.ProofreadKind_PROOFREAD_KIND_SCAN:
		options, ok := msg.GetOptions().(*numenv1.ProofreadTextRequest_Scan)
		if !ok {
			return textResult{}, connect.NewError(connect.CodeInvalidArgument, errScanOptions)
		}
		result, err = proofreadScan(ctx, s.ask, groups, options.Scan.GetMaxEditDistance())
	case numenv1.ProofreadKind_PROOFREAD_KIND_SPEECH:
		options, ok := msg.GetOptions().(*numenv1.ProofreadTextRequest_Speech)
		if !ok {
			return textResult{}, connect.NewError(connect.CodeInvalidArgument, errSpeechOptions)
		}
		speech := options.Speech
		if speech.GetSize() <= 0 {
			return textResult{}, connect.NewError(connect.CodeInvalidArgument, errSpeechSize)
		}
		var lines []Line
		for _, group := range groups {
			lines = append(lines, group.Lines...)
		}
		result, err = proofreadSpeech(ctx, s.ask, lines,
			int(speech.GetSize()), int(speech.GetOverlap()), int(speech.GetFromLine()), int(speech.GetLineLimit()))
	default:
		return textResult{}, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("unknown kind %s", msg.GetKind()))
	}
	if err != nil {
		return textResult{}, readModelError(ctx, err)
	}
	return result, nil
}

var (
	errScanOptions   = errors.New("a scan request carries scan options")
	errSpeechOptions = errors.New("a speech request carries speech options")
	errSpeechSize    = errors.New("a speech request holds at least one line to a model request")
)

// readModelError maps a failure to reach the model to the code the caller reads
// it by: a model nothing is routed to answer for is unavailable, and the end of
// the caller's context is its own.
func readModelError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return connect.NewError(connect.CodeCanceled, ctxErr)
	}
	if errors.Is(err, ai.ErrNoRoute) || errors.Is(err, ai.ErrUnavailable) || errors.Is(err, ai.ErrNoHost) {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

// readGroups returns the batches a request carries.
func readGroups(groups []*numenv1.ProofreadGroup) []Batch {
	out := make([]Batch, len(groups))
	for i, group := range groups {
		lines := make([]Line, len(group.GetLines()))
		for j, line := range group.GetLines() {
			lines[j] = Line{Number: int(line.GetNumber()), Text: line.GetText()}
		}
		out[i] = Batch{Number: int(group.GetNumber()), Lines: lines}
	}
	return out
}

func main() {
	proof := newProofreadingServer(ai.Complete)
	s := server.New()
	s.Register(numenv1connect.NewProofreadingServiceHandler(proof, s.HandlerOptions()...))

	if err := s.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
