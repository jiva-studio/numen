package settings_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
)

const (
	writerPathEnv = "NUMEN_SETTINGS_WRITER_PATH"
	writerNameEnv = "NUMEN_SETTINGS_WRITER_NAME"
	writerRuns    = 150
	pluginOwner   = "md.numen.sample"
)

// TestTheWriterProcessWritesKeysInALoop is the body of the second process that the tests below
// start, standing in for the command line: it does nothing unless the environment names a file.
func TestTheWriterProcessWritesKeysInALoop(t *testing.T) {
	path := os.Getenv(writerPathEnv)
	if path == "" {
		t.Skip("runs only as the second process of another test")
	}
	if err := writeKeysInLoop(settings.NewFiles(), path, os.Getenv(writerNameEnv)); err != nil {
		t.Fatal(err)
	}
}

func keyOf(name string, run int) string { return name + strconv.Itoa(run) }

func writeKeysInLoop(files *settings.Files, path, name string) error {
	for run := range writerRuns {
		one := settings.Setting{At: []string{pluginOwner, keyOf(name, run)}, Written: run}
		if err := files.Save(path, newDocument(), one); err != nil {
			return err
		}
	}
	return nil
}

func getMissingKeys(t *testing.T, path string, names ...string) []string {
	t.Helper()
	cfg, err := openAt(path)
	if err != nil {
		t.Fatal(err)
	}
	var missing []string
	for _, name := range names {
		for run := range writerRuns {
			if _, has := cfg.Plugins[pluginOwner][keyOf(name, run)]; !has {
				missing = append(missing, keyOf(name, run))
			}
		}
	}
	return missing
}

func TestTwoWritersOfOneFileEachKeepEveryKeyOfTheOther(t *testing.T) {
	path := write(t, "{}\n")
	var group sync.WaitGroup
	errs := make([]error, 2)
	// Two roots share no Files, as two processes do not.
	for i, name := range []string{"a", "b"} {
		group.Go(func() {
			errs[i] = writeKeysInLoop(settings.NewFiles(), path, name)
		})
	}
	group.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if missing := getMissingKeys(t, path, "a", "b"); len(missing) > 0 {
		t.Fatalf("%d keys were lost, first %q", len(missing), missing[0])
	}
}

func TestTwoProcessesWritingDifferentKeysInALoopKeepBothWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "numen.json")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	outputs := make([][]byte, 2)
	errs := make([]error, 2)
	for i, name := range []string{"a", "b"} {
		group.Go(func() {
			cmd := exec.CommandContext(t.Context(), exe, "-test.run=^TestTheWriterProcessWritesKeysInALoop$")
			cmd.Env = append(os.Environ(), writerPathEnv+"="+path, writerNameEnv+"="+name)
			outputs[i], errs[i] = cmd.CombinedOutput()
		})
	}
	group.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("writer %d: %v: %s", i, err, outputs[i])
		}
	}
	if missing := getMissingKeys(t, path, "a", "b"); len(missing) > 0 {
		t.Fatalf("%d keys were lost, first %q", len(missing), missing[0])
	}
}
