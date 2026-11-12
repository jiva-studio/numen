package service

import "testing"

func TestTheProviderFillsEveryRole(t *testing.T) {
	provider := newTestService(t).Provider()
	if provider.Completer == nil || provider.Embedder == nil || provider.Models == nil || provider.Profiles == nil {
		t.Errorf("provider = %+v, want every role filled", provider)
	}
}
