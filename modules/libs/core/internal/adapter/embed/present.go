package embed

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gomlx/go-huggingface/hub"
)

// IsFetched says whether this model's files are on this machine: in the folder
// the configuration names, or in the repository's place in the download cache.
//
// It stats what a fetch would have written, and opens, reaches and creates
// nothing.
func IsFetched(cfg LocalModel) bool {
	file := cfg.File
	if file == "" {
		file = ModelFile
	}
	if cfg.Dir != "" {
		return here(filepath.Join(cfg.Dir, file)) &&
			here(filepath.Join(cfg.Dir, TokenizerFile))
	}
	if cfg.Name == "" {
		return false
	}
	// A repository's files stand under the snapshot of the revision fetched.
	at := filepath.Join(
		hub.DefaultCacheDir(), folder(cfg.Name), "snapshots", "*", ModelFolder, file,
	)
	found, err := filepath.Glob(at)
	return err == nil && len(found) > 0
}

// folder is what one repository is called in the cache: the type it is, and its
// name with every part spelled out.
func folder(name string) string {
	parts := append([]string{string(hub.RepoTypeModel)}, strings.Split(name, "/")...)
	return strings.Join(parts, hub.RepoIdSeparator)
}

func here(at string) bool {
	_, err := os.Stat(at)
	return err == nil
}
