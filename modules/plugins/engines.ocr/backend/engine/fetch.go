package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

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

// Pins maps the address of a model file to the sha256 its bytes must hold.
type Pins map[string]string

// GetSum returns the sha256 pinned for an address.
func (p Pins) GetSum(address string) string {
	return p[address]
}

// Source is where a plugin looks for a model file and how it fetches one.
type Source struct {
	GetPaths func(file string) []string
	Fetch    func(ctx context.Context, address, sum string) (string, error)
}

// FindLocal returns the file of a model in one of the places the source names.
func FindLocal(src Source, pins Pins, name string) (string, bool) {
	sum := pins.GetSum(name)
	for _, p := range src.GetPaths(filepath.Base(name)) {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if sum == "" || VerifyFile(p, sum) == nil {
			return p, true
		}
	}
	return "", false
}

// Locate returns where one model's file is located.
func Locate(ctx context.Context, src Source, pins Pins, path, name, kind string) (string, error) {
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("the %s: %w", kind, err)
		}
		return path, nil
	}
	if name == "" {
		return "", fmt.Errorf("no %s: name one, or say where it is", kind)
	}
	if foundPath, ok := FindLocal(src, pins, name); ok {
		return foundPath, nil
	}
	if !IsAddress(name) {
		return "", fmt.Errorf("the %s %q is not beside the application, and is not somewhere to fetch it from", kind, name)
	}
	found, err := src.Fetch(ctx, name, pins.GetSum(name))
	if err != nil {
		return "", fmt.Errorf("the %s: %w", kind, err)
	}
	return found, nil
}

// Wanted is a model file to locate.
type Wanted struct {
	Dst  *string
	Path string
	Name string
	Kind string
}

// LocateAll locates every wanted model file into its destination.
func LocateAll(ctx context.Context, src Source, pins Pins, wanted []Wanted) error {
	for _, one := range wanted {
		at, err := Locate(ctx, src, pins, one.Path, one.Name, one.Kind)
		if err != nil {
			return err
		}
		if one.Dst != nil {
			*one.Dst = at
		}
	}
	return nil
}

// Folder is where a plugin keeps its model files.
type Folder struct {
	Section        string
	Dir            string
	ShouldDownload bool
	Progress       func(what string, done, total int64)
}

// NewFolder returns a Folder for a settings section.
func NewFolder(section, dir string, shouldDownload bool, progress func(what string, done, total int64)) Folder {
	return Folder{
		Section:        section,
		Dir:            dir,
		ShouldDownload: shouldDownload,
		Progress:       progress,
	}
}

// GetDir returns the folder model files are placed into.
func (f Folder) GetDir() (string, error) {
	if f.Dir != "" {
		return f.Dir, nil
	}
	return GetHome()
}

// Fetch downloads a model if not present.
func (f Folder) Fetch(ctx context.Context, address, sum string) (string, error) {
	dir, err := f.GetDir()
	if err != nil {
		return "", err
	}
	return Fetch(ctx, Request{
		Dir:            dir,
		Section:        f.Section,
		Address:        address,
		Sum:            sum,
		ShouldDownload: f.ShouldDownload,
		Progress:       f.Progress,
	})
}

// GetPaths returns candidate paths for a model file.
func (f Folder) GetPaths(file string) []string {
	var out []string
	if f.Dir != "" {
		out = append(out, filepath.Join(f.Dir, file))
	}
	if self, err := os.Executable(); err == nil {
		out = append(out, filepath.Join(filepath.Dir(self), file))
		out = append(out, filepath.Join(filepath.Dir(self), "models", file))
	}
	return out
}

// Source returns the search paths and download function.
func (f Folder) Source() Source {
	return Source{GetPaths: f.GetPaths, Fetch: f.Fetch}
}

// Locate finds and fetches each wanted model file.
func (f Folder) Locate(ctx context.Context, pins Pins, wanted []Wanted) error {
	return LocateAll(ctx, f.Source(), pins, wanted)
}

const (
	headerTimeout     = 2 * time.Minute
	minBytesPerSecond = 192 << 10
	unknownSizeBytes  = 1 << 30
)

func getDownloadTimeout(size int64) time.Duration {
	if size <= 0 {
		size = unknownSizeBytes
	}
	return headerTimeout + time.Duration(size/minBytesPerSecond)*time.Second
}

var client = &http.Client{
	CheckRedirect: func(request *http.Request, via []*http.Request) error {
		if request.URL.Scheme != "https" {
			return fmt.Errorf("%s redirects to %s", via[len(via)-1].URL, request.URL)
		}
		if len(via) >= 10 {
			return fmt.Errorf("%s redirects ten times", via[0].URL)
		}
		return nil
	},
}

// IsAddress reports whether name is an https address.
func IsAddress(name string) bool {
	return strings.HasPrefix(name, "https://")
}

// GetCacheName returns the filename an address is stored under.
func GetCacheName(address, sum string) string {
	base := getBaseName(address)
	if sum != "" {
		return sum[:min(len(sum), 12)] + "-" + base
	}
	digest := sha256.Sum256([]byte(address))
	return hex.EncodeToString(digest[:6]) + "-" + base
}

func getBaseName(address string) string {
	name := address
	if u, err := url.Parse(address); err == nil {
		name = u.Path
	}
	base := path.Base(name)
	if base == "" || base == "." || base == "/" {
		return "model.onnx"
	}
	return base
}

// VerifyFile checks whether the file matches the expected sha256.
func VerifyFile(name, sum string) error {
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if found := hex.EncodeToString(hash.Sum(nil)); found != strings.ToLower(sum) {
		return fmt.Errorf("%s holds %s, and %s is pinned", name, found, sum)
	}
	return nil
}

// Request is one file to locate or download into Dir.
type Request struct {
	Dir            string
	Section        string
	Address        string
	Sum            string
	ShouldDownload bool
	Progress       func(what string, done, total int64)
}

func (r Request) reportProgress(done, total int64) {
	if r.Progress != nil {
		r.Progress(getBaseName(r.Address), done, total)
	}
}

// Fetch downloads Request into Dir if absent or corrupt.
func Fetch(ctx context.Context, r Request) (string, error) {
	targetPath := filepath.Join(r.Dir, GetCacheName(r.Address, r.Sum))
	var corrupt error
	if _, err := os.Stat(targetPath); err == nil {
		if r.Sum == "" {
			return targetPath, nil
		}
		if corrupt = VerifyFile(targetPath, r.Sum); corrupt == nil {
			return targetPath, nil
		}
	}
	if err := checkCanDownload(r, corrupt); err != nil {
		return "", err
	}
	if err := os.MkdirAll(r.Dir, 0o755); err != nil {
		return "", err
	}
	if corrupt != nil {
		if err := os.Remove(targetPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
	}
	return targetPath, download(ctx, r, targetPath)
}

func checkCanDownload(r Request, corrupt error) error {
	if !r.ShouldDownload {
		if corrupt != nil {
			return corrupt
		}
		return fmt.Errorf("%s is not on this machine: set %s.dir to where it is, or %s.download to fetch it",
			r.Address, r.Section, r.Section)
	}
	if IsOffline() {
		if corrupt != nil {
			return corrupt
		}
		return fmt.Errorf("%s is not on this machine, and %s forbids fetching it", r.Address, EnvOffline)
	}
	return nil
}

func download(ctx context.Context, r Request, targetPath string) error {
	if !IsAddress(r.Address) {
		return fmt.Errorf("%s is not an https address", r.Address)
	}
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	waited := time.AfterFunc(headerTimeout, stop)
	defer waited.Stop()
	r.reportProgress(0, 0)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, r.Address, nil)
	if err != nil {
		return err
	}
	answer, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", r.Address, err)
	}
	defer answer.Body.Close()
	if answer.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching %s: %s", r.Address, answer.Status)
	}

	file, err := os.CreateTemp(r.Dir, filepath.Base(targetPath)+".part-*")
	if err != nil {
		return err
	}
	part := file.Name()
	waited.Reset(getDownloadTimeout(answer.ContentLength))

	if err := write(file, answer.Body, r); err != nil {
		_ = os.Remove(part)
		return fmt.Errorf("fetching %s: %w", r.Address, err)
	}
	if err := os.Rename(part, targetPath); err != nil {
		_ = os.Remove(part)
		return err
	}
	return nil
}

func write(file *os.File, body io.Reader, r Request) error {
	hash := sha256.New()
	counted := &progressWriter{
		to:         io.MultiWriter(file, hash),
		total:      0,
		onProgress: r.reportProgress,
	}
	_, err := io.Copy(counted, body)
	if err == nil && r.Sum != "" {
		if found := hex.EncodeToString(hash.Sum(nil)); found != strings.ToLower(r.Sum) {
			err = fmt.Errorf("download holds %s, and %s is pinned", found, r.Sum)
		}
	}
	if err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}

type progressWriter struct {
	to         io.Writer
	total      int64
	downloaded int64
	last       time.Time
	onProgress func(done, total int64)
}

func (c *progressWriter) Write(raw []byte) (int, error) {
	n, err := c.to.Write(raw)
	c.downloaded += int64(n)
	if now := time.Now(); now.Sub(c.last) > time.Second {
		c.last = now
		c.onProgress(c.downloaded, c.total)
	}
	return n, err
}
