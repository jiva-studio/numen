package transcription

import (
	"context"
	"os"
	"path/filepath"
)

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
