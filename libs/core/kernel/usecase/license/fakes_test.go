package license_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/license"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/licenseport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/license"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

type fakeDeclarations struct {
	plugins []manifest.Discovered
}

func (f *fakeDeclarations) GetPlugins() []manifest.Discovered {
	return f.plugins
}

type fakeKeyStore struct {
	mu   sync.Mutex
	keys map[string]string
}

func newFakeKeyStore() *fakeKeyStore {
	return &fakeKeyStore{keys: make(map[string]string)}
}

func (s *fakeKeyStore) GetKey(_ context.Context, name string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	val, ok := s.keys[name]
	if !ok {
		return "", plugindomain.ErrNoKey
	}
	return val, nil
}

func (s *fakeKeyStore) SetKey(_ context.Context, name, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[name] = key
	return nil
}

func (s *fakeKeyStore) DeleteKey(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.keys[name]; !ok {
		return plugindomain.ErrNoKey
	}
	delete(s.keys, name)
	return nil
}

type fakeMachine struct {
	fingerprint  string
	instanceName string
}

func (m fakeMachine) GetMachineFingerprint() string { return m.fingerprint }
func (m fakeMachine) GetInstanceName() string       { return m.instanceName }

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
}

type fakeChangeHub struct {
	mu    sync.Mutex
	count int
	ch    chan struct{}
}

func newFakeChangeHub() *fakeChangeHub {
	return &fakeChangeHub{ch: make(chan struct{}, 8)}
}

func (h *fakeChangeHub) NotifyChange(_ context.Context) {
	h.mu.Lock()
	h.count++
	h.mu.Unlock()
	select {
	case h.ch <- struct{}{}:
	default:
	}
}

func (h *fakeChangeHub) Subscribe() (<-chan struct{}, func()) {
	return h.ch, func() {}
}

type fakeValidator struct {
	mu             sync.Mutex
	activateCalls  int
	validateCalls  int
	deactivateErr  error
	activateResult licenseport.ActivationResult
	activateErr    error
	validateResult licenseport.ValidationResult
	validateErr    error
}

func (v *fakeValidator) ListProviderOptions(providers []manifest.LicenseProvider) []license.ProviderOption {
	var out []license.ProviderOption
	for _, p := range providers {
		out = append(out, license.ProviderOption{
			Provider:       p.Provider,
			ProviderName:   p.Provider,
			ProductID:      p.ProductID,
			CheckoutURL:    "https://example.com/buy/" + p.ProductID,
			PaymentMethods: []string{"Cards"},
		})
	}
	return out
}

func (v *fakeValidator) Activate(_ context.Context, _ []manifest.LicenseProvider, _, _ string) (licenseport.ActivationResult, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.activateCalls++
	return v.activateResult, v.activateErr
}

func (v *fakeValidator) Validate(_ context.Context, _, _, _ string) (licenseport.ValidationResult, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.validateCalls++
	return v.validateResult, v.validateErr
}

func (v *fakeValidator) Deactivate(_ context.Context, _, _, _ string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.deactivateErr
}
