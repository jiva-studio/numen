package settings

import (
	"reflect"
	"sync"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
)

// HostSubscriptions keeps the delivery of setting changes to the plugin hosts, one per host. A
// host is identified by the pointer the capabilities that reach it are built from.
type HostSubscriptions struct {
	hosts sync.Map
}

// hostSubscription is the delivery of setting changes to one plugin host.
type hostSubscription struct {
	mu          sync.Mutex
	unsubscribe func()
	ended       chan struct{}
}

func isKeyed(host any) bool {
	return host != nil && reflect.TypeOf(host).Comparable()
}

// Subscribe delivers each batch of setting changes to deliver, once per host. A host that is
// already subscribed is left as it is.
func (s *HostSubscriptions) Subscribe(
	host any, subscribe settingsport.ChangeSubscriber, deliver func([]settingsdomain.Setting),
) {
	var one *hostSubscription
	if isKeyed(host) {
		one = &hostSubscription{}
		one.mu.Lock()
		defer one.mu.Unlock()
		if _, isSubscribed := s.hosts.LoadOrStore(host, one); isSubscribed {
			return
		}
	}
	changes, unsubscribe := subscribe()
	ended := make(chan struct{})
	go func() {
		defer close(ended)
		for batch := range changes {
			deliver(batch)
		}
	}()
	if one != nil {
		one.unsubscribe, one.ended = unsubscribe, ended
	}
}

// Unsubscribe ends the delivery to the host and returns after the batch being delivered has been
// handled.
func (s *HostSubscriptions) Unsubscribe(host any) {
	if !isKeyed(host) {
		return
	}
	found, isSubscribed := s.hosts.LoadAndDelete(host)
	if !isSubscribed {
		return
	}
	one, isHostSubscription := found.(*hostSubscription)
	if !isHostSubscription {
		return
	}
	one.mu.Lock()
	defer one.mu.Unlock()
	one.unsubscribe()
	<-one.ended
}
