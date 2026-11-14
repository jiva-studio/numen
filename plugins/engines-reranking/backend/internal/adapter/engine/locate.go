package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gomlx/go-huggingface/hub"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
)

const (
	// EnvDir names the folder plugins keep their models in.
	EnvDir = "NUMEN_MODELS_DIR"
	// EnvOffline forbids fetching a model when set.
	EnvOffline = "NUMEN_OFFLINE"
)

// tokenizerFile is the tokenizer of a model, at the root of its repository.
const tokenizerFile = "tokenizer.json"

// ErrNotOnMachine is returned by Locate for a model this machine does not hold while EnvOffline forbids
// fetching it.
var ErrNotOnMachine = errors.New("the model is not on this machine")

// FetchProgress reports progress while downloading models.
type FetchProgress func(done, total int64)

// Paths are the model and tokenizer file locations.
type Paths struct {
	Model     string
	Tokenizer string
}

// Locate returns the files of the model of p from the download cache, fetching them at the pinned revision
// first where the cache does not hold them. A file that misses its pinned sha256 is deleted and is an error.
// progress, when not nil, receives the bytes fetched so far while a download runs.
func Locate(ctx context.Context, p profile.Profile, progress FetchProgress) (Paths, error) {
	home, err := getHome()
	if err != nil {
		return Paths{}, fmt.Errorf("models folder: %w", err)
	}
	repo := hub.New(p.ID).WithProgressBar(false).WithCacheDir(filepath.Join(home, "huggingface"))
	if p.Revision != "" {
		repo = repo.WithRevision(p.Revision)
	}
	if found, ok, err := locateCachedModel(repo, p); err != nil || ok {
		return found, err
	}
	if isOffline() {
		return Paths{}, fmt.Errorf("%w: %s, and %s forbids fetching it", ErrNotOnMachine, p.ID, EnvOffline)
	}
	return downloadModel(ctx, repo, p, progress)
}

func getHome() (string, error) {
	if dir := os.Getenv(EnvDir); dir != "" {
		return dir, nil
	}
	base, err := cache.GetSharedDataDirOrPluginDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "models"), nil
}

func isOffline() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvOffline))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func downloadModel(ctx context.Context, repo *hub.Repo, p profile.Profile, progress FetchProgress) (Paths, error) {
	sizes, err := listRepoFiles(repo)
	if err != nil {
		return Paths{}, fmt.Errorf("listing %s: %w", p.ID, err)
	}
	if _, ok := sizes[p.OnnxFile]; !ok {
		return Paths{}, fmt.Errorf("%s publishes no %s", p.ID, p.OnnxFile)
	}
	if _, ok := sizes[tokenizerFile]; !ok {
		return Paths{}, fmt.Errorf("%s publishes no %s", p.ID, tokenizerFile)
	}
	files := append(selectModelFiles(sizes, p.OnnxFile), tokenizerFile)
	if dir, err := repo.CacheDir(); err == nil {
		defer reportProgress(dir, files, sizes, progress)()
	}
	found := Paths{Model: "", Tokenizer: ""}
	for _, name := range files {
		local, err := repo.DownloadFileCtx(ctx, name)
		if err != nil {
			return Paths{}, fmt.Errorf("fetching %s: %w", name, err)
		}
		if err := checkPinnedChecksum(p.SHA256, name, local); err != nil {
			return Paths{}, err
		}
		switch name {
		case p.OnnxFile:
			found.Model = local
		case tokenizerFile:
			found.Tokenizer = local
		}
	}
	return found, nil
}

func listRepoFiles(repo *hub.Repo) (map[string]int64, error) {
	sizes := map[string]int64{}
	for info, err := range repo.IterFileInfos() {
		if err != nil {
			return nil, err
		}
		sizes[info.Name] = info.Size
	}
	return sizes, nil
}

// selectModelFiles returns the model file and the files beside it whose names start with its name, such as
// the external data of model.onnx in model.onnx_data.
func selectModelFiles(sizes map[string]int64, onnxFile string) []string {
	out := []string{onnxFile}
	for _, name := range slices.Sorted(maps.Keys(sizes)) {
		if name != onnxFile && path.Dir(name) == path.Dir(onnxFile) && strings.HasPrefix(name, onnxFile) {
			out = append(out, name)
		}
	}
	return out
}

func reportProgress(dir string, files []string, sizes map[string]int64, progress FetchProgress) func() {
	var total int64
	wanted := make(map[string]int64, len(files))
	for _, name := range files {
		wanted[path.Base(name)] = sizes[name]
		total += sizes[name]
	}
	if progress == nil || total <= 0 {
		return func() {}
	}
	isDone := make(chan struct{})
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-isDone:
				return
			case <-tick.C:
				progress(min(sumBytes(dir, wanted), total), total)
			}
		}
	}()
	return func() {
		close(isDone)
		<-doneCh
	}
}

func sumBytes(dir string, wanted map[string]int64) int64 {
	var sum int64
	err := filepath.WalkDir(dir, func(file string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		size, ours := wanted[strings.TrimSuffix(filepath.Base(file), ".incomplete")]
		if !ours {
			return nil
		}
		if info, err := os.Stat(file); err == nil {
			sum += min(info.Size(), size)
		}
		return nil
	})
	if err != nil {
		return 0
	}
	return sum
}

func verifyFile(file, wantSum string) (err error) {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if found := hex.EncodeToString(h.Sum(nil)); !strings.EqualFold(found, wantSum) {
		return fmt.Errorf("%s holds %s, and %s is pinned", file, found, wantSum)
	}
	return nil
}

func checkPinnedChecksum(sums map[string]string, name, file string) error {
	sum, ok := sums[name]
	if !ok {
		return nil
	}
	if err := verifyFile(file, sum); err != nil {
		return errors.Join(fmt.Errorf("%s: %w", name, err), discardDownload(file))
	}
	return nil
}

func discardDownload(file string) error {
	target, err := filepath.EvalSymlinks(file)
	if err != nil {
		target = file
	}
	var removed []error
	for _, one := range []string{file, target} {
		if err := os.Remove(one); err != nil && !errors.Is(err, fs.ErrNotExist) {
			removed = append(removed, err)
		}
		if target == file {
			break
		}
	}
	return errors.Join(removed...)
}

func locateCachedModel(repo *hub.Repo, p profile.Profile) (Paths, bool, error) {
	dir, err := repo.CacheDir()
	if err != nil {
		return Paths{}, false, fmt.Errorf("cache folder of %s: %w", p.ID, err)
	}
	if p.Revision == "" {
		return locateNewestSnapshot(dir, p)
	}
	found := snapshotPaths(filepath.Join(dir, "snapshots", p.Revision), p.OnnxFile)
	if !hasFile(found.Model) || !hasFile(found.Tokenizer) {
		return Paths{}, false, nil
	}
	if err := checkPinnedChecksum(p.SHA256, p.OnnxFile, found.Model); err != nil {
		return Paths{}, false, err
	}
	if err := checkPinnedChecksum(p.SHA256, tokenizerFile, found.Tokenizer); err != nil {
		return Paths{}, false, err
	}
	return found, true, nil
}

func locateNewestSnapshot(dir string, p profile.Profile) (Paths, bool, error) {
	snapshots := filepath.Join(dir, "snapshots")
	entries, err := os.ReadDir(snapshots)
	if errors.Is(err, fs.ErrNotExist) {
		return Paths{}, false, nil
	}
	if err != nil {
		return Paths{}, false, fmt.Errorf("snapshots of %s: %w", p.ID, err)
	}
	var newest Paths
	var newestTime time.Time
	isFound := false
	for _, entry := range entries {
		found := snapshotPaths(filepath.Join(snapshots, entry.Name()), p.OnnxFile)
		if !hasFile(found.Model) || !hasFile(found.Tokenizer) {
			continue
		}
		info, err := os.Stat(found.Model)
		if err != nil {
			continue
		}
		if !isFound || info.ModTime().After(newestTime) {
			newest, newestTime, isFound = found, info.ModTime(), true
		}
	}
	return newest, isFound, nil
}

func snapshotPaths(snapshot, onnxFile string) Paths {
	return Paths{
		Model:     filepath.Join(snapshot, filepath.FromSlash(onnxFile)),
		Tokenizer: filepath.Join(snapshot, tokenizerFile),
	}
}

func hasFile(name string) bool {
	info, err := os.Stat(name)
	return err == nil && !info.IsDir()
}
