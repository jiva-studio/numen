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

// readyName is the marker file whose presence means the folder next to it contains a complete dependency.
const readyName = ".ready"

var unsafeNameChars = regexp.MustCompile(`[^A-Za-z0-9._-]`)

func getDependencyDir(root, kind, id, version string) string {
	if version == "" {
		version = "latest"
	}
	return filepath.Join(root, cleanName(kind), cleanName(id), fmt.Sprintf("%s-%s-%s", cleanName(version), runtime.GOOS, runtime.GOARCH))
}

func cleanName(name string) string {
	clean := unsafeNameChars.ReplaceAllString(name, "_")
	if strings.Trim(clean, ".") == "" {
		return "_"
	}
	return clean
}

func isReady(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, readyName))
	return err == nil
}

func writeReadyMarker(ctx context.Context, dir, primary, sum string) error {
	tmp := filepath.Join(dir, readyName+".tmp")
	if err := os.WriteFile(tmp, []byte(primary+"\n"+sum+"\n"), 0o644); err != nil {
		return err
	}
	return renameWithRetry(ctx, tmp, filepath.Join(dir, readyName), newDefaultBackoff())
}

func readReadyMarker(dir string) (name, sum string, err error) {
	raw, err := os.ReadFile(filepath.Join(dir, readyName))
	if err != nil {
		return "", "", err
	}
	name, sum, _ = strings.Cut(strings.TrimSpace(string(raw)), "\n")
	return strings.TrimSpace(name), strings.TrimSpace(sum), nil
}

func isPlainName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, "/\\*?[\x00")
}

func isReserved(name string) bool {
	return strings.HasPrefix(name, readyName) || strings.HasPrefix(name, tempPrefix)
}

// backoff is the waits between the attempts at one operation.
type backoff struct {
	attempts int
	first    time.Duration
	sleep    func(context.Context, time.Duration) error
}

func newDefaultBackoff() backoff {
	return backoff{attempts: 6, first: 50 * time.Millisecond, sleep: sleepContext}
}

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

func isAccessDenied(err error) bool {
	return runtime.GOOS == "windows" && errors.Is(err, fs.ErrPermission)
}

func renameWithRetry(ctx context.Context, from, to string, policy backoff) error {
	return retry(ctx, policy, isAccessDenied, func() error { return os.Rename(from, to) })
}
