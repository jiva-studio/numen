package modelcalltest

import (
	"context"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
)

// Recorded is one call a provider received.
type Recorded struct {
	Operation  string
	Address    string
	Call       modelcall.ProviderCall
	Payload    []byte
	Operations []string
	IsFresh    bool
}

// Provider is a connectionport.Provider whose answers are scripted by the address of the connection.
// A call with no script sends no data frame and answers the done payload "answer from <address>".
type Provider struct {
	mu       sync.Mutex
	queued   map[string][]error
	always   map[string]error
	blocked  map[string]chan struct{}
	scripts  map[string]script
	listings map[string]modelcall.ListedOptions
	recorded []Recorded
	entered  chan string
}

// NewProvider creates a provider that answers every call.
func NewProvider() *Provider {
	return &Provider{
		queued: map[string][]error{}, always: map[string]error{}, blocked: map[string]chan struct{}{}, scripts: map[string]script{},
		listings: map[string]modelcall.ListedOptions{},
		entered:  make(chan string, 256),
	}
}

// Fail makes the next calls to the address fail with the errors before any frame, one for each call.
func (p *Provider) Fail(address string, errs ...error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queued[address] = append(p.queued[address], errs...)
}

// FailAlways makes every call to the address fail with err before any frame.
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

type script struct {
	frames []string
	err    error
}

// Script makes the calls to the address send the frames and then return err, or the done payload
// "answer from <address>" where err is nil.
func (p *Provider) Script(address string, err error, frames ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.scripts[address] = script{frames: frames, err: err}
}

// Call sends the frames scripted for the address. A failure scripted with Fail or FailAlways is returned
// before any frame.
func (p *Provider) Call(
	ctx context.Context, operation string, call modelcall.ProviderCall, payload []byte, send modelcall.Send,
) ([]byte, error) {
	address := call.Fields[addressField]
	if err := p.record(ctx, Recorded{Operation: operation, Address: address, Call: call, Payload: payload}); err != nil {
		return nil, err
	}
	p.mu.Lock()
	scripted := p.scripts[address]
	p.mu.Unlock()
	for _, one := range scripted.frames {
		if err := send([]byte(one)); err != nil {
			return nil, err
		}
	}
	if scripted.err != nil {
		return nil, scripted.err
	}
	return []byte("answer from " + address), nil
}

// ScriptOptions makes the listings of options of the address answer listed.
func (p *Provider) ScriptOptions(address string, listed modelcall.ListedOptions) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.listings[address] = listed
}

// ListOptions answers what ScriptOptions set for the address, and else two models, the group of the first
// naming the operations of the setting.
func (p *Provider) ListOptions(
	ctx context.Context, operation string, call modelcall.ProviderCall, query modelcall.OptionsQuery,
) (modelcall.ListedOptions, error) {
	address := call.Fields[addressField]
	recorded := Recorded{Operation: operation, Address: address, Call: call, Operations: query.Operations, IsFresh: query.IsFresh}
	if err := p.record(ctx, recorded); err != nil {
		return modelcall.ListedOptions{}, err
	}
	p.mu.Lock()
	listed, isScripted := p.listings[address]
	p.mu.Unlock()
	if isScripted {
		return listed, nil
	}
	return modelcall.ListedOptions{Options: []modelcall.Option{
		{Value: "model-1", Title: "Model one", Group: strings.Join(query.Operations, " ")}, {Value: "model-2"},
	}}, nil
}

// ListDetected answers that nothing is found; it is not recorded as a call.
func (p *Provider) ListDetected(context.Context, modelcall.ProviderCall) ([]modelcall.DetectedConnection, error) {
	return nil, nil
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
