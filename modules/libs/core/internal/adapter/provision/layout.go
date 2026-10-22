package provision

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// EnvDir names the folder dependencies are kept in.
const EnvDir = "NUMEN_DEPENDENCIES_DIR"

// readyName is the file whose presence says the folder beside it holds a whole dependency.
const readyName = ".ready"

var unsafeNameChars = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// GetRoot returns the folder dependencies are kept in: EnvDir, or
// .numen/dependencies under the home folder.
func GetRoot() (string, error) {
	if dir := os.Getenv(EnvDir); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".numen", "dependencies"), nil
}

// getDependencyDir returns the folder one version of a dependency is kept in on
// this platform: <root>/<kind>/<id>/<version>-<os>-<arch>.
func getDependencyDir(root, kind, id, version string) string {
	if version == "" {
		version = "latest"
	}
	return filepath.Join(root, cleanName(kind), cleanName(id), fmt.Sprintf("%s-%s-%s", cleanName(version), runtime.GOOS, runtime.GOARCH))
}

// cleanName returns the name as one safe path element.
func cleanName(name string) string {
	clean := unsafeNameChars.ReplaceAllString(name, "_")
	if strings.Trim(clean, ".") == "" {
		return "_"
	}
	return clean
}

// IsReady reports whether the folder holds a whole dependency.
func isReady(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, readyName))
	return err == nil
}

// writeReady marks the folder as holding a whole dependency. The sentinel names
// the primary file.
func writeReady(ctx context.Context, dir, primary string) error {
	tmp := filepath.Join(dir, readyName+".tmp")
	if err := os.WriteFile(tmp, []byte(primary), 0o644); err != nil {
		return err
	}
	return renameWithRetry(ctx, tmp, filepath.Join(dir, readyName), defaultBackoff)
}

// readReady returns the primary file the sentinel names.
func readReady(dir string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(dir, readyName))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

// backoff is the waits between the attempts at one operation.
type backoff struct {
	attempts int
	first    time.Duration
	sleep    func(context.Context, time.Duration) error
}

var defaultBackoff = backoff{attempts: 6, first: 50 * time.Millisecond, sleep: sleepContext}

func sleepContext(ctx context.Context, wait time.Duration) error {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// retry runs the operation until it succeeds, fails with an error that isRetryable
// refuses, or the attempts run out. The wait doubles after each failure.
func retry(ctx context.Context, policy backoff, isRetryable func(error) bool, operation func() error) error {
	wait := policy.first
	var err error
	for attempt := 1; ; attempt++ {
		if err = operation(); err == nil || !isRetryable(err) || attempt >= policy.attempts {
			return err
		}
		if sleepErr := policy.sleep(ctx, wait); sleepErr != nil {
			return errors.Join(err, sleepErr)
		}
		wait *= 2
	}
}

// isAccessDenied reports whether the error is the platform refusing a file that
// another process still holds open, which on windows an antivirus scan or an
// indexer does for a moment after the file is written.
func isAccessDenied(err error) bool {
	return runtime.GOOS == "windows" && errors.Is(err, fs.ErrPermission)
}

// renameWithRetry moves the file to its place, trying again while the platform
// reports access denied.
func renameWithRetry(ctx context.Context, from, to string, policy backoff) error {
	return retry(ctx, policy, isAccessDenied, func() error { return os.Rename(from, to) })
}
