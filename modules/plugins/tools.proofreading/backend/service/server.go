// Package service is the backend of the proofreading plugin.
package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"strconv"
	"sync"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/ai"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// Default proofreading sizes.
const (
	DefaultBatchSize = 40
	DefaultOverlap   = 4
	DefaultInFlight  = 4
)

// purpose is what every question to the model is for. It routes nothing.
const purpose = "proofreading"

// completer asks the host for the answer to a question.
type completer func(ctx context.Context, question *numenv1.CompletionQuestion) (string, error)

type proofreadingConfig struct {
	batchSize       int
	overlap         int
	inFlight        int
	maxEditDistance float64
}

func defaultProofreadingConfig() proofreadingConfig {
	return proofreadingConfig{
		batchSize:       DefaultBatchSize,
		overlap:         DefaultOverlap,
		inFlight:        DefaultInFlight,
		maxEditDistance: MaxEditDistance,
	}
}

type proofreadingServer struct {
	ask askModel
	mu  sync.RWMutex
	cfg proofreadingConfig
}

// newProofreadingServer returns a server whose model is the one the host routes
// the proofreading purpose to.
func newProofreadingServer(complete completer) *proofreadingServer {
	return &proofreadingServer{
		ask: newAskModel(complete),
		cfg: defaultProofreadingConfig(),
	}
}

func (s *proofreadingServer) updateConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := values["batch_size"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			s.cfg.batchSize = n
		}
	}
	if v, ok := values["overlap"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			s.cfg.overlap = n
		}
	}
	if v, ok := values["in_flight"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			s.cfg.inFlight = n
		}
	}
	if v, ok := values["max_edit_distance"]; ok {
		if d, err := strconv.ParseFloat(v, 64); err == nil && d >= 0 {
			s.cfg.maxEditDistance = d
		}
	}
}

func (s *proofreadingServer) getConfig() proofreadingConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// newAskModel returns the model as a system instruction and a prompt are put to it.
func newAskModel(complete completer) askModel {
	return func(ctx context.Context, instruction, prompt string) (string, error) {
		return complete(ctx, &numenv1.CompletionQuestion{
			Purpose: purpose,
			Messages: []*numenv1.CompletionMessage{
				{Role: "system", Text: instruction},
				{Role: "user", Text: prompt},
			},
		})
	}
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
		maxDist := options.Scan.GetMaxEditDistance()
		if maxDist <= 0 {
			maxDist = s.getConfig().maxEditDistance
		}
		result, err = proofreadScan(ctx, s.ask, groups, maxDist)
	case numenv1.ProofreadKind_PROOFREAD_KIND_SPEECH:
		options, ok := msg.GetOptions().(*numenv1.ProofreadTextRequest_Speech)
		if !ok {
			return textResult{}, connect.NewError(connect.CodeInvalidArgument, errSpeechOptions)
		}
		cfg := s.getConfig()
		speech := options.Speech
		size := int(speech.GetSize())
		if size <= 0 {
			size = cfg.batchSize
		}
		overlap := int(speech.GetOverlap())
		if overlap <= 0 && speech.GetSize() <= 0 {
			overlap = cfg.overlap
		}
		fromLine := int(speech.GetFromLine())
		lineLimit := int(speech.GetLineLimit())
		if lineLimit <= 0 {
			lineLimit = cfg.inFlight * size
		}
		var lines []Line
		for _, group := range groups {
			lines = append(lines, group.Lines...)
		}
		result, err = proofreadSpeech(ctx, s.ask, lines, size, overlap, fromLine, lineLimit)
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
	errScanOptions   = errors.New("a scan request has scan options")
	errSpeechOptions = errors.New("a speech request has speech options")
)

// readModelError maps a failure to reach the model to the code the caller reads
// it by: a model nothing is routed to is unavailable, one the person has not let
// this plugin ask is permission denied, and the end of the caller's context is
// its own.
func readModelError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return connect.NewError(connect.CodeCanceled, ctxErr)
	}
	if errors.Is(err, ai.ErrNoConsent) {
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	if errors.Is(err, ai.ErrNoRoute) || errors.Is(err, ai.ErrUnavailable) || errors.Is(err, ai.ErrNoHost) {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

// readGroups returns the batches a request contains.
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

// Mount registers the proofreading service on s, asking the host named by the environment for each
// completion.
func Mount(s *server.Server) error {
	return mount(s, ai.Complete, nil)
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given. It asks the host those name.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		return mount(s, func(ctx context.Context, question *numenv1.CompletionQuestion) (string, error) {
			client, err := ai.NewClientFromVars(vars)
			if err != nil {
				return "", err
			}
			return client.Complete(ctx, question)
		}, vars)
	})
}

func mount(s *server.Server, complete completer, vars map[string]string) error {
	srv := newProofreadingServer(complete)
	if len(vars) > 0 {
		srv.updateConfig(vars)
	}
	s.SetUpdateConfig(srv.updateConfig)
	s.Register(numenv1connect.NewProofreadingServiceHandler(srv, s.HandlerOptions()...))
	return nil
}
