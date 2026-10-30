package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gomlx/go-huggingface/hub"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/clock"
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

const (
	// EnvDir names the folder plugins keep their models in.
	EnvDir = "NUMEN_MODELS_DIR"
	// EnvOffline forbids fetching a model when set.
	EnvOffline = "NUMEN_OFFLINE"
	homeDir    = "numen/models"
)

// GetHome returns the shared models folder.
func GetHome() (string, error) {
	if dir := os.Getenv(EnvDir); dir != "" {
		return dir, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, filepath.FromSlash(homeDir)), nil
}

// IsOffline reports whether EnvOffline forbids fetching.
func IsOffline() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvOffline))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func verifyFile(path, wantSum string) (err error) {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if found := hex.EncodeToString(h.Sum(nil)); !strings.EqualFold(found, wantSum) {
		return fmt.Errorf("%s holds %s, and %s is pinned", path, found, wantSum)
	}
	return nil
}

func downloadModel(ctx context.Context, cfg LocalModel, file string, progress FetchProgress) (Paths, error) {
	home, err := GetHome()
	if err != nil {
		return Paths{}, fmt.Errorf("models folder: %w", err)
	}
	repo := hub.New(cfg.Name).WithProgressBar(false).WithCacheDir(filepath.Join(home, "huggingface"))
	if cfg.Revision != "" {
		repo = repo.WithRevision(cfg.Revision)
	}
	if p, ok, err := locateCached(repo, cfg, file); err != nil || ok {
		return p, err
	}
	if IsOffline() {
		return Paths{}, fmt.Errorf("%s is not on this machine, and %s forbids fetching it", cfg.Name, EnvOffline)
	}
	folder, sizes, tokenizerSize, err := listRepoFiles(repo)
	if err != nil {
		return Paths{}, fmt.Errorf("listing %s: %w", cfg.Name, err)
	}
	if !slices.Contains(folder, file) {
		return Paths{}, fmt.Errorf("%s publishes no %s/%s: it has %v", cfg.Name, ModelFolder, file, folder)
	}

	files := selectModelFiles(folder, file)
	release := reserveDownload(repo, files, sizes, tokenizerSize, progress)
	defer release()

	p, err := fetchModelFiles(ctx, repo, files, file, cfg.Sums)
	if err != nil {
		return Paths{}, err
	}
	p.Tokenizer, err = fetchTokenizer(ctx, repo, cfg.Name, cfg.Sums)
	if err != nil {
		return Paths{}, err
	}
	return p, nil
}

func reserveDownload(
	repo *hub.Repo,
	files []string,
	sizes map[string]int64,
	tokenizerSize int64,
	progress FetchProgress,
) func() {
	var total int64
	for _, f := range files {
		total += sizes[f]
	}
	sizes[TokenizerFile] = tokenizerSize
	total += tokenizerSize
	counted := append(slices.Clone(files), TokenizerFile)

	dir, dirErr := repo.CacheDir()
	var cancelProgress func()
	if dirErr == nil {
		cancelProgress = reportProgress(dir, counted, sizes, total, progress)
	}
	return func() {
		if cancelProgress != nil {
			cancelProgress()
		}
	}
}

func fetchTokenizer(ctx context.Context, repo *hub.Repo, name string, sums map[string]string) (string, error) {
	if !repo.HasFile(TokenizerFile) {
		return "", fmt.Errorf("%s publishes no %s", name, TokenizerFile)
	}
	tokenizer, err := repo.DownloadFileCtx(ctx, TokenizerFile)
	if err != nil {
		return "", fmt.Errorf("fetching %s: %w", TokenizerFile, err)
	}
	if err := checkPinned(sums, TokenizerFile, tokenizer); err != nil {
		return "", err
	}
	return tokenizer, nil
}

func fetchModelFiles(ctx context.Context, repo *hub.Repo, files []string, file string, sums map[string]string) (Paths, error) {
	p := Paths{Model: "", Tokenizer: ""}
	for _, name := range files {
		pos, err := repo.DownloadFileCtx(ctx, ModelFolder+"/"+name)
		if err != nil {
			return Paths{}, fmt.Errorf("fetching %s/%s: %w", ModelFolder, name, err)
		}
		if err := checkPinned(sums, ModelFolder+"/"+name, pos); err != nil {
			return Paths{}, err
		}
		if name == file {
			p.Model = pos
		}
		if name == TokenizerFile {
			p.Tokenizer = pos
		}
	}
	return p, nil
}

// listRepoFiles returns the files of the model folder with their sizes, and the size of the tokenizer beside it.
func listRepoFiles(repo *hub.Repo) ([]string, map[string]int64, int64, error) {
	var out []string
	var tokenizer int64
	sizes := map[string]int64{}
	for info, err := range repo.IterFileInfos() {
		if err != nil {
			return nil, nil, 0, err
		}
		if info.Name == TokenizerFile {
			tokenizer = info.Size
		}
		rest, inside := strings.CutPrefix(info.Name, ModelFolder+"/")
		if !inside || rest == "" {
			continue
		}
		out = append(out, rest)
		sizes[rest] = info.Size
	}
	return out, sizes, tokenizer, nil
}

func reportProgress(dir string, files []string, sizes map[string]int64, total int64, progress FetchProgress) func() {
	if progress == nil || total <= 0 {
		return func() {}
	}
	isDone := make(chan struct{})
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		tick := clock.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-isDone:
				return
			case <-tick.C:
				progress(min(sumBytes(dir, files, sizes), total), total)
			}
		}
	}()
	return func() {
		close(isDone)
		<-doneCh
	}
}

func sumBytes(dir string, files []string, sizes map[string]int64) int64 {
	wanted := make(map[string]int64, len(files))
	for _, name := range files {
		wanted[name] = sizes[name]
	}
	var sum int64
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
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
	if err != nil {
		return 0
	}
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

func isFileOfAnyModel(name string, siblings []string) bool {
	for _, model := range siblings {
		if strings.HasPrefix(name, model) {
			return true
		}
	}
	return false
}

// checkPinned returns an error when the file at path holds other than the sum
// pinned for name. A name with no pin is not checked.
func checkPinned(sums map[string]string, name, path string) error {
	sum, ok := sums[name]
	if !ok {
		return nil
	}
	if err := verifyFile(path, sum); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// locateCached returns the files of a snapshot that the cache already holds. A
// model pinned to a revision takes that snapshot, each file checked against its
// pinned sum. A model with no revision takes the newest snapshot that holds both
// the model and the tokenizer.
func locateCached(repo *hub.Repo, cfg LocalModel, file string) (Paths, bool, error) {
	cache, err := repo.CacheDir()
	if err != nil {
		return Paths{}, false, fmt.Errorf("cache folder of %s: %w", cfg.Name, err)
	}
	if cfg.Revision == "" {
		return locateNewestSnapshot(cache, cfg.Name, file)
	}
	snapshot := filepath.Join(cache, "snapshots", cfg.Revision)
	p := snapshotPaths(snapshot, file)
	if !hasFile(p.Model) || !hasFile(p.Tokenizer) {
		return Paths{Model: "", Tokenizer: ""}, false, nil
	}
	if err := checkPinned(cfg.Sums, ModelFolder+"/"+file, p.Model); err != nil {
		return Paths{}, false, err
	}
	if err := checkPinned(cfg.Sums, TokenizerFile, p.Tokenizer); err != nil {
		return Paths{}, false, err
	}
	return p, true, nil
}

// locateNewestSnapshot returns the files of the most recently downloaded
// snapshot in cache that holds both the model and the tokenizer. The downloader
// links a file into a snapshot only after the download has finished.
func locateNewestSnapshot(cache, name, file string) (Paths, bool, error) {
	snapshots := filepath.Join(cache, "snapshots")
	entries, err := os.ReadDir(snapshots)
	if errors.Is(err, fs.ErrNotExist) {
		return Paths{Model: "", Tokenizer: ""}, false, nil
	}
	if err != nil {
		return Paths{}, false, fmt.Errorf("snapshots of %s: %w", name, err)
	}
	var newest Paths
	var newestTime time.Time
	isFound := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		p := snapshotPaths(filepath.Join(snapshots, entry.Name()), file)
		if !hasFile(p.Model) || !hasFile(p.Tokenizer) {
			continue
		}
		info, err := os.Stat(p.Model)
		if err != nil {
			continue
		}
		if !isFound || info.ModTime().After(newestTime) {
			newest, newestTime, isFound = p, info.ModTime(), true
		}
	}
	return newest, isFound, nil
}

// snapshotPaths returns where the model and the tokenizer lie in a snapshot.
func snapshotPaths(snapshot, file string) Paths {
	return Paths{
		Model:     filepath.Join(snapshot, ModelFolder, file),
		Tokenizer: filepath.Join(snapshot, TokenizerFile),
	}
}

func hasFile(name string) bool {
	info, err := os.Stat(name)
	return err == nil && !info.IsDir()
}
