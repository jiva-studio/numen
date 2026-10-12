package recognition

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
)

// isolate points everything a test could write to at a folder of its own.
func isolate(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
}

const freshProcess = "NUMEN_RECOGNITION_FRESH_PROCESS"

// runInFreshProcess runs one test again in a process that has opened no
// runtime, because the runtime is the process's and stays once opened.
func runInFreshProcess(t *testing.T, test string) {
	t.Helper()
	home := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^"+test+"$", "-test.v")
	cmd.Env = append(os.Environ(),
		freshProcess+"=1",
		"HOME="+home,
		"XDG_CACHE_HOME="+filepath.Join(home, "cache"),
		"XDG_CONFIG_HOME="+filepath.Join(home, "config"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s in a fresh process: %v\n%s", test, err, out)
	}
}

// isFreshProcess says whether this process has opened no runtime, so that the
// test asking is the first to.
func isFreshProcess() bool {
	return !onnxruntime.IsAvailable(onnxruntime.Settings{Runtime: filepath.Join(os.TempDir(), "numen-no-such-runtime")})
}

// A runtime that cannot be opened stops the search for models.
func TestLocateStopsWhenTheRuntimeCannotBeOpened(t *testing.T) {
	if !isFreshProcess() {
		runInFreshProcess(t, "TestLocateStopsWhenTheRuntimeCannotBeOpened")
		return
	}
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing-runtime.so")
	cfg := Config{
		Runtime:   missing,
		Dir:       dir,
		Layout:    LayoutModel{Path: writeFile(t, dir, "layout.onnx")},
		Detect:    DetectModel{Path: writeFile(t, dir, "detect.onnx")},
		Recognise: RecogniserModel{Path: writeFile(t, dir, "inference.onnx")},
	}

	opened, found, err := locate(t.Context(), cfg)
	if err == nil {
		t.Fatal("located models on a runtime that cannot be opened")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("error = %v, want it to name %s", err, missing)
	}
	if opened != (through{}) || found != (paths{}) {
		t.Errorf("located %+v, %+v", opened, found)
	}
}

// A path written down is used as given, and its absence is an error that says
// what was missing.
func TestAModelPathThatIsGoneIsAnError(t *testing.T) {
	isolate(t)
	gone := filepath.Join(t.TempDir(), "gone.onnx")

	got, err := model(t.Context(), Config{}, gone, "", "layout")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error = %v, want one wrapping %v", err, fs.ErrNotExist)
	}
	if err != nil && !strings.Contains(err.Error(), "layout") {
		t.Errorf("error = %v, want it to name the model", err)
	}
	if got != "" {
		t.Errorf("path = %q", got)
	}
}

func TestAModelPathThatIsThereIsUsedAsGiven(t *testing.T) {
	isolate(t)
	at := writeFile(t, t.TempDir(), "layout.onnx")

	got, err := model(t.Context(), Config{}, at, "ignored.onnx", "layout")
	if err != nil || got != at {
		t.Errorf("model = %q, %v; want %q", got, err, at)
	}
}

// A name is looked for in the models folder under the file it ends in.
func TestAModelNameIsFoundInTheModelsFolder(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	at := writeFile(t, dir, "found.onnx")

	got, err := model(t.Context(), Config{Dir: dir}, "", "https://example.invalid/models/found.onnx", "detect")
	if err != nil || got != at {
		t.Errorf("model = %q, %v; want %q", got, err, at)
	}
}

// A name that is not in the models folder is fetched only if it is an address,
// and only if fetching is allowed.
func TestAModelNameThatIsNowhereToBeFound(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	cfg := Config{Dir: dir}

	_, err := model(t.Context(), cfg, "", "elsewhere.onnx", "detect")
	if err == nil || !strings.Contains(err.Error(), "is not beside the application") {
		t.Errorf("a plain name answered with %v", err)
	}

	_, err = model(t.Context(), cfg, "", "https://example.invalid/models/elsewhere.onnx", "detect")
	if err == nil || strings.Contains(err.Error(), "is not beside the application") ||
		!strings.Contains(err.Error(), "is not on this machine") {
		t.Errorf("an address with fetching off answered with %v", err)
	}

	if _, err = model(t.Context(), cfg, "", "", "detect"); err == nil || !strings.Contains(err.Error(), "no detect model") {
		t.Errorf("no name answered with %v", err)
	}
}

// What stops a fetch is the cause it is said with.
func TestAModelThatCannotBeFetchedKeepsItsCause(t *testing.T) {
	isolate(t)
	file := writeFile(t, t.TempDir(), "not-a-folder")
	cfg := Config{Dir: filepath.Join(file, "models"), ShouldDownload: true}

	_, err := model(t.Context(), cfg, "", "https://example.invalid/models/elsewhere.onnx", "recognise")
	if !errors.Is(err, syscall.ENOTDIR) {
		t.Errorf("error = %v, want one wrapping %v", err, syscall.ENOTDIR)
	}
	if err != nil && !strings.Contains(err.Error(), "recognise") {
		t.Errorf("error = %v, want it to name the model", err)
	}
}
