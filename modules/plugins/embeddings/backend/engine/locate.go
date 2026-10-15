package engine

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

// FetchProgress reports progress while downloading models.
type FetchProgress func(done, total int64)

// Paths are the model and tokenizer file locations.
type Paths struct {
	Model     string
	Tokenizer string
}

// Locate finds the model files: in a specified directory, or in the download cache.
func Locate(ctx context.Context, cfg LocalModel, progress FetchProgress) (Paths, error) {
	file := cfg.File
	if file == "" {
		file = ModelFile
	}
	if cfg.Dir != "" {
		return locateInDir(cfg.Dir, file)
	}
	if cfg.Name == "" {
		return Paths{}, errors.New("no model to run: name a repository or a directory")
	}
	if !cfg.ShouldDownload {
		return Paths{}, fmt.Errorf("%s is not on this machine: set dir to where it is, or enable download", cfg.Name)
	}
	return downloadModel(ctx, cfg, file, progress)
}

func locateInDir(dir, file string) (Paths, error) {
	p := Paths{
		Model:     filepath.Join(dir, file),
		Tokenizer: filepath.Join(dir, TokenizerFile),
	}
	for _, required := range []string{p.Model, p.Tokenizer} {
		if _, err := os.Stat(required); err != nil {
			return Paths{}, fmt.Errorf("model directory %s: %w", dir, err)
		}
	}
	return p, nil
}

func downloadModel(ctx context.Context, cfg LocalModel, file string, progress FetchProgress) (Paths, error) {
	repo := hub.New(cfg.Name).WithProgressBar(false)
	folder, sizes, err := listRepoFiles(repo)
	if err != nil {
		return Paths{}, fmt.Errorf("listing %s: %w", cfg.Name, err)
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

	p, err := fetchModelFiles(ctx, repo, files, file)
	if err != nil {
		return Paths{}, err
	}
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

func fetchModelFiles(ctx context.Context, repo *hub.Repo, files []string, file string) (Paths, error) {
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
	return p, nil
}

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

func sumBytes(dir string, files []string, sizes map[string]int64) int64 {
	wanted := make(map[string]int64, len(files))
	for _, name := range files {
		wanted[name] = sizes[name]
	}
	var sum int64
	_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipDir
		}
		if entry.IsDir() {
			return nil
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

func selectModelFiles(folder []string, named string) []string {
	var others []string
	for _, name := range folder {
		if name != named && strings.HasSuffix(name, ".onnx") {
			others = append(others, name)
		}
	}
	out := []string{named}
	for _, name := range folder {
		if name == named || isFileOfAnyModel(name, others) {
			continue
		}
		out = append(out, name)
	}
	return out
}

func isFileOfAnyModel(name string, models []string) bool {
	for _, model := range models {
		if strings.HasPrefix(name, model) {
			return true
		}
	}
	return false
}
