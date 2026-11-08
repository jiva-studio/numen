package container

import "sync"

// pluginChanges tells each subscriber that the host applied a batch of settings or forgot the
// failure of a plugin. A subscriber that has not taken the last signal gets one signal for both.
type pluginChanges struct {
	mu          sync.Mutex
	next        int
	subscribers map[int]chan struct{}
}

func (p *pluginChanges) subscribe() (<-chan struct{}, func()) {
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

func (p *pluginChanges) signal() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, signals := range p.subscribers {
		select {
		case signals <- struct{}{}:
		default:
		}
	}
}

// WatchPluginChanges reports each time the plugin host applied a batch of settings, which enables,
// disables and configures plugins, and each time it forgot the failure of a plugin, as after a retry
// that succeeded. The returned function stops the reports.
func (c Config) WatchPluginChanges() (<-chan struct{}, func()) {
	return c.getResources().pluginChanges.subscribe()
}
