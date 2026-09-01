package transcription

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	ort "github.com/getcharzp/onnxruntime_purego"
)

// The runtime is published as one archive per platform rather than as a library
// on its own, so fetching it is fetching the archive and taking the one file out
// of it. What is inside is the same everywhere but its folder is named after the
// release, which is why the file is found by its name rather than by its path.
// The version is the one the binding asks the library for: it requests API 23,
// and a release older than 1.23 answers that it does not have it.
const runtimeVersion = "1.23.0"

// runtimeNames are what the library is called, in the order it is looked for.
var runtimeNames = []string{"libonnxruntime.so", "libonnxruntime.dylib", "onnxruntime.dll"}

// runtimeName is what the library is called on this machine.
func runtimeName() string {
	switch runtime.GOOS {
	case "darwin":
		return "libonnxruntime.dylib"
	case "windows":
		return "onnxruntime.dll"
	}
	return "libonnxruntime.so"
}

// runtimeAddress is where this platform's archive is published.
func runtimeAddress() (string, error) {
	const from = "https://github.com/microsoft/onnxruntime/releases/download/v" + runtimeVersion + "/"
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		return from + "onnxruntime-linux-x64-" + runtimeVersion + ".tgz", nil
	case "linux/arm64":
		return from + "onnxruntime-linux-aarch64-" + runtimeVersion + ".tgz", nil
	case "darwin/arm64":
		return from + "onnxruntime-osx-arm64-" + runtimeVersion + ".tgz", nil
	case "darwin/amd64":
		return from + "onnxruntime-osx-x86_64-" + runtimeVersion + ".tgz", nil
	case "windows/amd64":
		return from + "onnxruntime-win-x64-" + runtimeVersion + ".zip", nil
	}
	return "", fmt.Errorf("no onnx runtime is published for %s/%s: name one in transcription.runtime",
		runtime.GOOS, runtime.GOARCH)
}

// library is the ONNX Runtime this machine listens through, opened.
//
// A path in the settings is used as given. Otherwise what the machine already
// holds is tried, and only a machine holding none fetches one — a library that
// is here is a hundred and thirty megabytes nobody waits for.
func library(ctx context.Context, cfg Config) (*ort.Engine, string, error) {
	held.Lock()
	defer held.Unlock()
	if held.engine != nil {
		return held.engine, held.at, nil
	}

	if cfg.Runtime != "" {
		engine, err := ort.NewEngine(cfg.Runtime)
		if err != nil {
			return nil, "", fmt.Errorf("the onnx runtime %s: %w", cfg.Runtime, err)
		}
		return keep(engine, cfg.Runtime)
	}

	support()
	engine, at, refused := opened(present(cfg))
	if engine != nil {
		return keep(engine, at)
	}

	address, err := runtimeAddress()
	if err != nil {
		return nil, "", err
	}
	archive, err := fetched(ctx, cfg, address, cfg.Download)
	if err != nil {
		return nil, "", fmt.Errorf("the onnx runtime: %w", err)
	}
	at, err = unpacked(archive, runtimeName())
	if err != nil {
		return nil, "", err
	}
	if engine, err = ort.NewEngine(at); err != nil {
		refused = append(refused, fmt.Sprintf("%s: %v", at, err))
		return nil, "", fmt.Errorf("no onnx runtime this machine opens — name one in transcription.runtime:\n  %s",
			strings.Join(refused, "\n  "))
	}
	return keep(engine, at)
}

// held is the runtime this process listens with, and where it came from.
//
// One for the life of the process, made before the window. The lock is over the
// opening, which two transcriptions may reach at once.
var held struct {
	sync.Mutex
	engine *ort.Engine
	at     string
}

// here says whether this process has its runtime. It is read without the lock,
// so it is answered while another transcription is opening one.
var here atomic.Bool

// keep is the runtime this process has settled on. The lock is the caller's.
func keep(engine *ort.Engine, at string) (*ort.Engine, string, error) {
	held.engine, held.at = engine, at
	here.Store(true)
	return engine, at, nil
}

// present is every ONNX Runtime this machine holds without fetching one: beside
// the application, wherever this platform keeps its packages, and by the name
// the loader searches for on its own.
func present(cfg Config) []string {
	var out []string
	for _, at := range beside(cfg.Dir, runtimeNames...) {
		if _, err := os.Stat(at); err == nil {
			out = append(out, at)
		}
	}
	out = append(out, installed(runtimeName(), "*-onnxruntime-*")...)
	return append(out, runtimeName())
}

// opened is the first library that loads, the one that did, and why each of the
// others would not. A file that is there is not a library that loads.
func opened(candidates []string) (*ort.Engine, string, []string) {
	var refused []string
	for _, at := range candidates {
		engine, err := ort.NewEngine(at)
		if err == nil {
			return engine, at, nil
		}
		refused = append(refused, fmt.Sprintf("%s: %v", at, err))
	}
	return nil, "", refused
}

// directories are where this machine keeps shared libraries.
func directories() []string {
	out := filepath.SplitList(os.Getenv("LD_LIBRARY_PATH"))
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, filepath.Join(home, ".nix-profile", "lib"))
	}
	return append(out,
		"/run/current-system/sw/lib",
		"/usr/local/lib",
		"/usr/lib64",
		"/usr/lib/"+runtime.GOARCH+"-linux-gnu",
		"/usr/lib/x86_64-linux-gnu",
		"/usr/lib/aarch64-linux-gnu",
		"/usr/lib",
		"/lib64",
		"/lib",
	)
}

// installed is every file on this machine named for one library, on the
// loader's own path and in the package store beside it. The store is searched
// through the package's name because a machine that has one holds nothing on
// the loader's path at all.
func installed(name, pkg string) []string {
	var out []string
	for _, dir := range directories() {
		out = append(out, matching(filepath.Join(dir, name+"*"))...)
	}
	return append(out, matching(filepath.Join("/nix/store", pkg, "lib", name+"*"))...)
}

// matching is the regular files one pattern names.
func matching(pattern string) []string {
	found, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	var out []string
	for _, at := range found {
		if info, err := os.Stat(at); err == nil && info.Mode().IsRegular() {
			out = append(out, at)
		}
	}
	return out
}

// unpacked takes the library out of an archive, once, and says where it now is.
//
// It is looked for by name at any depth, because the folder inside the archive
// is named after the release. The plain name inside is a link to the file
// carrying the version, so what is taken out is the file: a link copied out of
// an archive points at nothing.
func unpacked(archive, name string) (string, error) {
	at := filepath.Join(filepath.Dir(archive), name)
	if _, err := os.Stat(at); err == nil {
		return at, nil
	}
	if strings.HasSuffix(archive, ".zip") {
		return at, fromZip(archive, name, at)
	}
	return at, fromTgz(archive, name, at)
}

func fromTgz(archive, name, at string) error {
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	unzipped, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("%s: %w", archive, err)
	}
	defer unzipped.Close()

	held := tar.NewReader(unzipped)
	for {
		entry, err := held.Next()
		if err == io.EOF {
			return fmt.Errorf("%s holds no %s", archive, name)
		}
		if err != nil {
			return err
		}
		if entry.Typeflag != tar.TypeReg || !isLibrary(path.Base(entry.Name), name) {
			continue
		}
		return write(at, held)
	}
}

func fromZip(archive, name, at string) error {
	held, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer held.Close()
	for _, entry := range held.File {
		if !isLibrary(path.Base(entry.Name), name) {
			continue
		}
		file, err := entry.Open()
		if err != nil {
			return err
		}
		defer file.Close()
		return write(at, file)
	}
	return fmt.Errorf("%s holds no %s", archive, name)
}

// write puts one file down through a name beside it, so that an unpacking
// interrupted leaves nothing that looks finished.
func write(at string, from io.Reader) error {
	part := at + ".part"
	file, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, from); err != nil {
		file.Close()
		os.Remove(part)
		return err
	}
	if err := file.Close(); err != nil {
		os.Remove(part)
		return err
	}
	return os.Rename(part, at)
}

// isLibrary says whether one name in an archive is the library.
//
// It is the plain name or that name with a version after it, and it is not one
// of the libraries published beside it: those are named for what they add, so
// they carry the same prefix and a different word.
func isLibrary(found, name string) bool {
	return found == name || strings.HasPrefix(found, name+".")
}
