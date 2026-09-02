package transcription

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
)

// standing says whether the runtime a recording is heard through was made.
var standing atomic.Bool

// Prepared says whether this process made the runtime before it made anything
// else.
func Prepared() bool { return standing.Load() }

// Prepare makes the runtime a recording is heard through, and is called before
// a window is. Every recording this process hears is heard through it, and one
// made after a window says nothing about every recording it is given.
//
// Nothing is fetched: a machine that holds no runtime says so, and listening is
// what fetches one.
func Prepare(ctx context.Context, cfg Config) error {
	cfg.Download = false
	if _, _, err := library(ctx, cfg); err != nil {
		return err
	}
	standing.Store(true)
	return nil
}

// Ready says whether everything a transcription needs is already on this
// machine. It opens nothing and fetches nothing, so it is answered while a
// transcription is fetching what it needs.
func Ready(cfg Config) bool {
	cfg.Download = false
	if !runtimeHere(cfg) {
		return false
	}
	var found paths
	for _, one := range wanted(cfg, &found) {
		if _, err := model(context.Background(), cfg, one.path, one.name, one.what); err != nil {
			return false
		}
	}
	return true
}

// runtimeHere says whether this process has its runtime, or this machine holds
// a file to make one from. A name the loader would search for on its own is not
// a file anything here can find.
func runtimeHere(cfg Config) bool {
	if here.Load() {
		return true
	}
	if cfg.Runtime != "" {
		_, err := os.Stat(cfg.Runtime)
		return err == nil
	}
	for _, at := range present(cfg) {
		if filepath.IsAbs(at) {
			return true
		}
	}
	return false
}
