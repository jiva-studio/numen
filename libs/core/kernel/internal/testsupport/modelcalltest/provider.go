package modelcalltest

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
)

// Recorded is one call a provider received.
type Recorded struct {
	Operation string
	Address   string
	Call      modelcall.ProviderCall
	Request   modelcall.CompleteRequest
	Chat      modelcall.ChatRequest
}

// Provider is a connectionport.Provider whose answers are scripted by the address of the connection.
// A call with no script succeeds.
type Provider struct {
	mu        sync.Mutex
	queued    map[string][]error
	always    map[string]error
	blocked   map[string]chan struct{}
	scripts   map[string]chatScript
	recorded  []Recorded
	entered   chan string
	IsUnknown bool
}

// NewProvider creates a provider that answers every call.
func NewProvider() *Provider {
	return &Provider{
		queued: map[string][]error{}, always: map[string]error{}, blocked: map[string]chan struct{}{}, scripts: map[string]chatScript{},
		entered: make(chan string, 256),
	}
}

// Fail makes the next calls to the address fail with the errors, one for each call.
func (p *Provider) Fail(address string, errs ...error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queued[address] = append(p.queued[address], errs...)
}

// FailAlways makes every call to the address fail with err.
func (p *Provider) FailAlways(address string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.always[address] = err
}

// Block makes every call to the address wait until its context ends or Unblock is called.
func (p *Provider) Block(address string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.blocked[address] = make(chan struct{})
}

// Calls returns the calls received, in order.
func (p *Provider) Calls() []Recorded {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]Recorded(nil), p.recorded...)
}

// CallsTo returns the calls received for the address.
func (p *Provider) CallsTo(address string) []Recorded {
	var calls []Recorded
	for _, one := range p.Calls() {
		if one.Address == address {
			calls = append(calls, one)
		}
	}
	return calls
}

func (p *Provider) receive(ctx context.Context, operation string, call modelcall.ProviderCall, request ...modelcall.CompleteRequest) error {
	recorded := Recorded{Operation: operation, Address: call.Fields["baseUrl"], Call: call}
	if len(request) > 0 {
		recorded.Request = request[0]
	}
	return p.record(ctx, recorded)
}

func (p *Provider) record(ctx context.Context, recorded Recorded) error {
	address := recorded.Address
	p.mu.Lock()
	p.recorded = append(p.recorded, recorded)
	block := p.blocked[address]
	err := p.always[address]
	if queue := p.queued[address]; err == nil && len(queue) > 0 {
		err, p.queued[address] = queue[0], queue[1:]
	}
	p.mu.Unlock()
	select {
	case p.entered <- address:
	default:
	}
	if block != nil {
		select {
		case <-block:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}

// Complete answers "answer from <address>".
func (p *Provider) Complete(
	ctx context.Context, call modelcall.ProviderCall, request modelcall.CompleteRequest,
) (modelcall.CompleteResult, error) {
	if err := p.receive(ctx, "complete", call, request); err != nil {
		return modelcall.CompleteResult{}, err
	}
	return modelcall.CompleteResult{Text: "answer from " + call.Fields["baseUrl"]}, nil
}

// Embed answers a vector of two numbers for each text: its length and 1.
func (p *Provider) Embed(ctx context.Context, call modelcall.ProviderCall, texts []string) ([][]float32, error) {
	if err := p.receive(ctx, "embed", call); err != nil {
		return nil, err
	}
	vectors := make([][]float32, len(texts))
	for i, text := range texts {
		vectors[i] = []float32{float32(len(text)), 1}
	}
	return vectors, nil
}

// ListModels answers two models, the first one labeled with the operation.
func (p *Provider) ListModels(ctx context.Context, call modelcall.ProviderCall, operation string) ([]modelcall.Model, error) {
	if err := p.receive(ctx, "list models", call); err != nil {
		return nil, err
	}
	return []modelcall.Model{{ID: "model-1", Title: "Model one", Operations: []string{operation}}, {ID: "model-2"}}, nil
}

// ListReasoningLevels answers two levels.
func (p *Provider) ListReasoningLevels(ctx context.Context, call modelcall.ProviderCall) ([]modelcall.ReasoningLevel, error) {
	if err := p.receive(ctx, "list reasoning levels", call); err != nil {
		return nil, err
	}
	return []modelcall.ReasoningLevel{{Value: "low", Title: "Low"}, {Value: "high", Title: "High"}}, nil
}

// GetModelProfile answers a profile of width 3, or an unknown profile where IsUnknown is set.
func (p *Provider) GetModelProfile(ctx context.Context, call modelcall.ProviderCall) (modelcall.ModelProfile, error) {
	if err := p.receive(ctx, "get model profile", call); err != nil {
		return modelcall.ModelProfile{}, err
	}
	if p.IsUnknown {
		return modelcall.ModelProfile{}, nil
	}
	return modelcall.ModelProfile{
		IsKnown: true, Dimensions: 3, MaxTokens: 8191, Pooling: "mean", QueryPrefix: "query: ", DocumentPrefix: "passage: ",
		Source: call.Type + " " + call.Fields["baseUrl"],
	}, nil
}

// ProbeEmbeddingWidth answers 7.
func (p *Provider) ProbeEmbeddingWidth(ctx context.Context, call modelcall.ProviderCall) (int, error) {
	if err := p.receive(ctx, "probe embedding width", call); err != nil {
		return 0, err
	}
	return 7, nil
}

type chatScript struct {
	events []modelcall.ChatEvent
	err    error
}

// ScriptChat makes the calls of Chat to the address send the events and then return err. A call with no
// script sends "answer from <address>" and a finish of reason stop.
func (p *Provider) ScriptChat(address string, err error, events ...modelcall.ChatEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.scripts[address] = chatScript{events: events, err: err}
}

// Chat sends the events scripted for the address. A failure scripted with Fail or FailAlways is returned
// before any event.
func (p *Provider) Chat(
	ctx context.Context, call modelcall.ProviderCall, request modelcall.ChatRequest, send func(modelcall.ChatEvent) error,
) error {
	address := call.Fields["baseUrl"]
	if err := p.record(ctx, Recorded{Operation: "chat", Address: address, Call: call, Chat: request}); err != nil {
		return err
	}
	p.mu.Lock()
	script, isScripted := p.scripts[address]
	p.mu.Unlock()
	if !isScripted {
		stop := modelcall.ChatFinishStop
		script = chatScript{events: []modelcall.ChatEvent{{TextDelta: "answer from " + address}, {Finish: &stop}}}
	}
	for _, one := range script.events {
		if err := send(one); err != nil {
			return err
		}
	}
	return script.err
}

// Entered receives the address of each call as the provider takes it.
func (p *Provider) Entered() <-chan string { return p.entered }

// Unblock answers the calls waiting at the address and lets the next ones answer at once.
func (p *Provider) Unblock(address string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if block, isBlocked := p.blocked[address]; isBlocked {
		close(block)
		delete(p.blocked, address)
	}
}

// Release answers the calls waiting at the address now and keeps the address blocked for
// the calls that come later.
func (p *Provider) Release(address string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if block, isBlocked := p.blocked[address]; isBlocked {
		close(block)
		p.blocked[address] = make(chan struct{})
	}
}
