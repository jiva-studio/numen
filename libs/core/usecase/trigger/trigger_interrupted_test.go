package trigger_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/usecase/trigger"
)

// closingPlugins ends the pass, as a vault closed part-way does, once the first run has started.
type closingPlugins struct {
	*plugins
	cancel context.CancelFunc
}

func (p closingPlugins) RunCommand(ctx context.Context, call pluginport.PluginCommandCall) (pluginport.PluginCommandResult, error) {
	result, err := p.plugins.RunCommand(ctx, call)
	p.cancel()
	return result, err
}

func TestARunOfAPassEndedPartWayIsNotRunAgain(t *testing.T) {
	vault := newMemoryVault(map[string]string{"scans/a.pdf": "%PDF a", "scans/b.pdf": "%PDF b"})
	ran := &plugins{triggers: []pluginport.PluginTriggerEntry{getExtractOnAdd()}, failing: map[string]bool{}}
	record := &memoryRecord{}
	config := func() (map[string]any, error) { return map[string]any{}, nil }
	ctx, cancel := context.WithCancel(t.Context())
	first := trigger.NewTriggers(closingPlugins{plugins: ran, cancel: cancel}, readers{vault}, types{}, record, config)
	if err := first.Execute(ctx, domain.Vault{ID: "v1"}); err == nil {
		t.Fatal("a pass whose context ended returned no error")
	}
	if got, want := ran.take(), []string{scanner + " app:file.createText scans/a.pdf"}; !slices.Equal(got, want) {
		t.Fatalf("the pass ended part-way ran %v, want %v", got, want)
	}

	next := trigger.NewTriggers(ran, readers{vault}, types{}, record, config)
	if err := next.Execute(t.Context(), domain.Vault{ID: "v1"}); err != nil {
		t.Fatal(err)
	}
	if got, want := ran.take(), []string{scanner + " app:file.createText scans/b.pdf"}; !slices.Equal(got, want) {
		t.Errorf("the next pass ran %v, want %v", got, want)
	}
}
