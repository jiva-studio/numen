package main

import (
	"io"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/webui"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// serving says whether the tools are in front of the agents.
func (r *reaching) serving() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.shut != nil
}

// windowOn is a window open on a vault of this test's own.
func windowOn(t *testing.T) (*webui.Opened, container.Config) {
	t.Helper()

	cfg := container.Config{
		IndexPath:    filepath.Join(t.TempDir(), "index.db"),
		RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
	}
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if _, err := filesystem.Initialize(root, filesystem.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}
	add := usecase.Add{Identity: cfg.VaultIdentity(), Registry: registry, Now: time.Now}
	if _, err := add.Execute(root, "one"); err != nil {
		t.Fatal(err)
	}

	opened, err := webui.Open(t.Context(), cfg, "one", io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { opened.Close() })
	return opened, cfg
}

// TestOneSwapHoldsTheAgentsUntilItIsOver. The tools are served for the vault in
// the window, so a second swap arriving while one runs does not put them back
// in front of the agents on the vault that is going.
func TestOneSwapHoldsTheAgentsUntilItIsOver(t *testing.T) {
	opened, cfg := windowOn(t)
	r := &reaching{
		ctx:    t.Context(),
		cfg:    cfg,
		opened: opened,
		opts:   agentOptions{off: true},
		out:    io.Discard,
	}
	r.on()
	if !r.serving() {
		t.Fatal("the tools were never served")
	}

	running := make(chan struct{})
	release := make(chan struct{})

	var swaps sync.WaitGroup
	swaps.Add(1)
	go func() {
		defer swaps.Done()
		_ = r.around(func() error {
			close(running)
			<-release
			return nil
		})
	}()
	<-running

	second := make(chan struct{})
	swaps.Add(1)
	go func() {
		defer swaps.Done()
		defer close(second)
		_ = r.around(func() error { return nil })
	}()
	// The second swap has this long to reach the endpoint the first is holding.
	select {
	case <-second:
	case <-time.After(time.Second):
	}
	served := r.serving()

	close(release)
	swaps.Wait()

	if served {
		t.Error("the tools were served again while a swap was running")
	}
	if !r.serving() {
		t.Error("the tools were not served again once the swap was over")
	}
}
