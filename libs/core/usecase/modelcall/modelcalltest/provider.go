package modelcalltest

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/core/port/connectionport"
)

// Recorded is one call a provider received.
type Recorded struct {
	Operation string
	Address   string
	Call      connectionport.ProviderCall
	Request   connectionport.CompleteRequest
}

// Provider is a connectionport.Provider whose answers are scripted by the address of the connection.
// A call with no script succeeds.
type Provider struct {
	mu        sync.Mutex
	queued    map[string][]error
	always    map[string]error
	blocked   map[string]bool
	recorded  []Recorded
	entered   chan string
	IsUnknown bool
}

// NewProvider creates a provider that answers every call.
func NewProvider() *Provider {
	return &Provider{
		queued: map[string][]error{}, always: map[string]error{}, blocked: map[string]bool{}, entered: make(chan string, 256),
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

// Block makes every call to the address wait until its context ends.
func (p *Provider) Block(address string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.blocked[address] = true
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

func (p *Provider) receive(ctx context.Context, operation string, call connectionport.ProviderCall, request ...connectionport.CompleteRequest) error {
	address := call.Fields["baseUrl"]
	p.mu.Lock()
	recorded := Recorded{Operation: operation, Address: address, Call: call}
	if len(request) > 0 {
		recorded.Request = request[0]
	}
	p.recorded = append(p.recorded, recorded)
	isBlocked := p.blocked[address]
	err := p.always[address]
	if queue := p.queued[address]; err == nil && len(queue) > 0 {
		err, p.queued[address] = queue[0], queue[1:]
	}
	p.mu.Unlock()
	select {
	case p.entered <- address:
	default:
	}
	if isBlocked {
		<-ctx.Done()
		return ctx.Err()
	}
	return err
}

// Complete answers "answer from <address>".
func (p *Provider) Complete(
	ctx context.Context, call connectionport.ProviderCall, request connectionport.CompleteRequest,
) (connectionport.CompleteResult, error) {
	if err := p.receive(ctx, "complete", call, request); err != nil {
		return connectionport.CompleteResult{}, err
	}
	return connectionport.CompleteResult{Text: "answer from " + call.Fields["baseUrl"]}, nil
}

// Embed answers a vector of two numbers for each text: its length and 1.
func (p *Provider) Embed(ctx context.Context, call connectionport.ProviderCall, texts []string) ([][]float32, error) {
	if err := p.receive(ctx, "embed", call); err != nil {
		return nil, err
	}
	vectors := make([][]float32, len(texts))
	for i, text := range texts {
		vectors[i] = []float32{float32(len(text)), 1}
	}
	return vectors, nil
}

// ListModels answers two models, the first one labelled with the operation.
func (p *Provider) ListModels(ctx context.Context, call connectionport.ProviderCall, operation string) ([]connectionport.Model, error) {
	if err := p.receive(ctx, "list models", call); err != nil {
		return nil, err
	}
	return []connectionport.Model{{ID: "model-1", Title: "Model one", Operations: []string{operation}}, {ID: "model-2"}}, nil
}

// ListReasoningLevels answers two levels.
func (p *Provider) ListReasoningLevels(ctx context.Context, call connectionport.ProviderCall) ([]connectionport.ReasoningLevel, error) {
	if err := p.receive(ctx, "list reasoning levels", call); err != nil {
		return nil, err
	}
	return []connectionport.ReasoningLevel{{Value: "low", Title: "Low"}, {Value: "high", Title: "High"}}, nil
}

// GetModelProfile answers a profile of width 3, or an unknown profile where IsUnknown is set.
func (p *Provider) GetModelProfile(ctx context.Context, call connectionport.ProviderCall) (connectionport.ModelProfile, error) {
	if err := p.receive(ctx, "get model profile", call); err != nil {
		return connectionport.ModelProfile{}, err
	}
	if p.IsUnknown {
		return connectionport.ModelProfile{}, nil
	}
	return connectionport.ModelProfile{
		IsKnown: true, Dimensions: 3, MaxTokens: 8191, Pooling: "mean", QueryPrefix: "query: ", DocumentPrefix: "passage: ",
		Source: call.Type + " " + call.Fields["baseUrl"],
	}, nil
}

// ProbeEmbeddingWidth answers 7.
func (p *Provider) ProbeEmbeddingWidth(ctx context.Context, call connectionport.ProviderCall) (int, error) {
	if err := p.receive(ctx, "probe embedding width", call); err != nil {
		return 0, err
	}
	return 7, nil
}

// Entered receives the address of each call as the provider takes it.
func (p *Provider) Entered() <-chan string { return p.entered }

// Unblock lets the calls to the address answer again.
func (p *Provider) Unblock(address string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.blocked, address)
}
