// Package pluginchanges tells each subscriber that the plugin host applied a batch of settings or
// forgot the failure of a plugin.
package pluginchanges

import "sync"

// Signals tells each subscriber of a change. A subscriber that has not taken the last signal gets one
// signal for both. The zero value is ready.
type Signals struct {
	mu          sync.Mutex
	next        int
	subscribers map[int]chan struct{}
}

// Subscribe returns the signals of the subscriber and the function that ends the subscription.
func (p *Signals) Subscribe() (<-chan struct{}, func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.subscribers == nil {
		p.subscribers = map[int]chan struct{}{}
	}
	id := p.next
	p.next++
	signals := make(chan struct{}, 1)
	p.subscribers[id] = signals
	return signals, func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		delete(p.subscribers, id)
	}
}

// Signal tells every subscriber of a change.
func (p *Signals) Signal() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, signals := range p.subscribers {
		select {
		case signals <- struct{}{}:
		default:
		}
	}
}
