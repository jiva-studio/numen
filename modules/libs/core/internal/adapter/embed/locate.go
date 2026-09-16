package embed

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gomlx/go-huggingface/hub"
)

// FetchProgress is how much of a model is here and how much is wanted, told while it
// comes down. Nothing is told for a model that is already on this machine.
type FetchProgress func(done, total int64)

// Paths are the files a model is made of. tokenizer.json defines the tokeniser
// in full, special tokens included.
type Paths struct {
	Model     string
	Tokenizer string
}

// Locate finds the model's files: in a directory the configuration names, or in
// the download cache. A named directory is what an installation with no network
// uses.
//
// The files are the same whichever engine runs them, so both embedders come
// here for them.
func Locate(ctx context.Context, cfg LocalModel, progress FetchProgress) (Paths, error) {
	file := cfg.File
	if file == "" {
		file = ModelFile
	}
	if cfg.Dir != "" {
		p := Paths{
			Model:     filepath.Join(cfg.Dir, file),
			Tokenizer: filepath.Join(cfg.Dir, TokenizerFile),
		}
		for _, required := range []string{p.Model, p.Tokenizer} {
			if _, err := os.Stat(required); err != nil {
				return Paths{}, fmt.Errorf("model directory %s: %w", cfg.Dir, err)
			}
		}
		return p, nil
	}
	if cfg.Name == "" {
		return Paths{}, errors.New("no model to run: name a repository or a directory")
	}
	if !cfg.Download {
		return Paths{}, fmt.Errorf("%s is not on this machine: set local.dir to where it is, or local.download to fetch it", cfg.Name)
	}

	repo := hub.New(cfg.Name).WithProgressBar(false)
	folder, sizes, err := listRepoFiles(repo)
	if err != nil {
		return Paths{}, fmt.Errorf("what %s publishes: %w", cfg.Name, err)
	}
	if !slices.Contains(folder, file) {
		return Paths{}, fmt.Errorf("%s publishes no %s/%s: it has %v", cfg.Name, ModelFolder, file, folder)
	}

	files := selectModelFiles(folder, file)
	var total int64
	for _, name := range files {
		total += sizes[name]
	}
	if dir, err := repo.CacheDir(); err == nil {
		defer reportProgress(dir, files, sizes, total, progress)()
	}

	p := Paths{}
	for _, name := range files {
		at, err := repo.DownloadFileCtx(ctx, ModelFolder+"/"+name)
		if err != nil {
			return Paths{}, fmt.Errorf("fetching %s/%s: %w", ModelFolder, name, err)
		}
		if name == file {
			p.Model = at
		}
		if name == TokenizerFile {
			p.Tokenizer = at
		}
	}
	// The tokeniser stands at the root of a repository, and beside the models
	// in some.
	if repo.HasFile(TokenizerFile) {
		if p.Tokenizer, err = repo.DownloadFileCtx(ctx, TokenizerFile); err != nil {
			return Paths{}, fmt.Errorf("fetching %s: %w", TokenizerFile, err)
		}
	}
	if p.Tokenizer == "" {
		return Paths{}, fmt.Errorf("%s publishes no %s", cfg.Name, TokenizerFile)
	}
	return p, nil
}

// listRepoFiles is what a repository holds beside its models, by the names
// they have inside that folder, and how large each is.
func listRepoFiles(repo *hub.Repo) ([]string, map[string]int64, error) {
	var out []string
	sizes := map[string]int64{}
	for info, err := range repo.IterFileInfos() {
		if err != nil {
			return nil, nil, err
		}
		rest, inside := strings.CutPrefix(info.Name, ModelFolder+"/")
		if !inside || rest == "" {
			continue
		}
		out = append(out, rest)
		sizes[rest] = info.Size
	}
	return out, sizes, nil
}

// reportProgress reports how much of the model is on this machine while it
// comes down, and hands back what stops the reporting.
//
// What is counted is the bytes under the repository's own place in the cache,
// which is what has arrived.
func reportProgress(dir string, files []string, sizes map[string]int64, total int64, progress FetchProgress) func() {
	if progress == nil || total <= 0 {
		return func() {}
	}
	done := make(chan struct{})
	over := make(chan struct{})
	go func() {
		defer close(over)
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-done:
				return
			case <-tick.C:
				progress(min(sumBytes(dir, files, sizes), total), total)
			}
		}
	}()
	return func() {
		close(done)
		<-over
	}
}

// sumBytes is how many bytes of the files named stand under a folder. A folder
// holding another build of the same model holds bytes that are not this one's,
// and a file part-written counts for no more than the size it will take.
//
// A cache files one copy of a model and hangs its names off it, so a name is
// weighed as what it points at.
func sumBytes(dir string, files []string, sizes map[string]int64) int64 {
	wanted := make(map[string]int64, len(files))
	for _, name := range files {
		wanted[name] = sizes[name]
	}
	var sum int64
	_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			// A folder the walk could not read weighs nothing, and the sum
			// carries that as a count short of what is on disk.
			return nil //nolint:nilerr // what could not be read is carried out in the sum
		}
		was, ours := wanted[strings.TrimSuffix(filepath.Base(path), ".incomplete")]
		if !ours {
			return nil
		}
		if info, err := os.Stat(path); err == nil {
			sum += min(info.Size(), was)
		}
		return nil
	})
	return sum
}

// selectModelFiles is everything the model named is made of, out of what stands
// beside it: the model, and every file that is not another model's.
//
// A model too large for one file keeps its weights in a second under its own
// name, and a graph may point at a constant in a third. A folder holds the full
// build and the quantised ones together, and taking one means leaving the
// gigabytes belonging to the others.
func selectModelFiles(folder []string, named string) []string {
	var others []string
	for _, name := range folder {
		if name != named && strings.HasSuffix(name, ".onnx") {
			others = append(others, name)
		}
	}
	out := []string{named}
	for _, name := range folder {
		if name == named || theirs(name, others) {
			continue
		}
		out = append(out, name)
	}
	return out
}

// theirs says a file belongs to one of the models given: it is that model, or
// it stands beside it under that model's name.
func theirs(name string, models []string) bool {
	for _, model := range models {
		if strings.HasPrefix(name, model) {
			return true
		}
	}
	return false
}
