// Package launch starts the recognitions of files and keeps them running until they end or the
// launcher is closed.
package launch

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/recognition"
)

// The states a start reports.
const (
	StateStarted = "started"
	StateRunning = "running"
	StateDone    = "done"
	StateSkipped = "skipped"
)

var (
	errNoVault  = errors.New("the call names no vault")
	errNotFound = errors.New("the file is not found")
)

// Outcome is what a call to Create did for a file.
type Outcome struct {
	TaskID  string
	Path    string
	State   string
	Version string
}

// Launcher starts recognitions that run until they end or Close.
type Launcher struct {
	recognitions *recognition.Recognition
	pluginID     string

	wg sync.WaitGroup

	mu       sync.Mutex
	isClosed bool
	running  map[string]context.CancelFunc
	mode     settings.RecognizeMode
	skipped  map[skipKey]struct{}
}

// skipKey names one version of a file: a change of its size or modification time makes a new key.
type skipKey struct {
	vault   domain.VaultID
	path    string
	size    int64
	modTime time.Time
}

// New returns a launcher of the recognitions. pluginID names the plugin whose text versions count as
// the recognition of a file.
func New(recognitions *recognition.Recognition, pluginID string) *Launcher {
	return &Launcher{
		recognitions: recognitions,
		pluginID:     pluginID,
		wg:           sync.WaitGroup{},
		mu:           sync.Mutex{},
		isClosed:     false,
		running:      map[string]context.CancelFunc{},
		mode:         settings.ModeOff,
		skipped:      map[skipKey]struct{}{},
	}
}

// Close cancels the recognitions the launcher started and waits for them to end.
func (l *Launcher) Close() {
	l.mu.Lock()
	l.isClosed = true
	for _, cancel := range l.running {
		cancel()
	}
	l.mu.Unlock()
	l.wg.Wait()
}

// Create starts the recognition of a file of the vault unless one runs or the file is recognized.
func (l *Launcher) Create(ctx context.Context, vault domain.VaultID, path string) (Outcome, error) {
	if vault == "" {
		return Outcome{}, errNoVault
	}
	if path == "" {
		return Outcome{}, errors.New("path required")
	}
	if _, err := l.recognitions.StatFile(ctx, vault, path); err != nil {
		if errors.Is(err, host.ErrNotExist) {
			return Outcome{}, fmt.Errorf("%s: %w", path, errNotFound)
		}
		return Outcome{}, fmt.Errorf("find %s: %w", path, err)
	}
	out := Outcome{TaskID: recognition.GetTaskID(vault, path), Path: path, State: StateRunning, Version: ""}
	isRunning, err := l.recognitions.IsRunning(ctx, vault, path)
	if err != nil {
		return Outcome{}, err
	}
	if isRunning {
		return out, nil
	}
	version, isFound, err := l.recognitions.FindLiveDerivedVersion(ctx, vault, path, l.pluginID)
	if err != nil {
		return Outcome{}, err
	}
	if isFound {
		out.State, out.Version = StateDone, string(version.GetID())
		return out, nil
	}
	out.State, err = l.start(ctx, vault, path)
	if err != nil {
		return Outcome{}, err
	}
	return out, nil
}

// SetMode sets when StartMissingFile recognizes a file. A change of the mode makes the launcher forget the
// files it skipped.
func (l *Launcher) SetMode(mode settings.RecognizeMode) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if mode != l.mode {
		l.skipped = map[skipKey]struct{}{}
	}
	l.mode = mode
}

// StartFile starts the recognition of a file of the shown vault whatever the mode, and fails when one
// already runs.
func (l *Launcher) StartFile(ctx context.Context, path string) (string, error) {
	vault, _, err := l.statFile(ctx, path)
	if err != nil {
		return "", err
	}
	return l.startNew(ctx, vault, path)
}

// StartMissingFile starts the recognition of a file that has no text of its own, and returns
// StateSkipped for a file with a text layer under the mode whenNoTextLayer.
func (l *Launcher) StartMissingFile(ctx context.Context, path string) (string, error) {
	vault, stat, err := l.statFile(ctx, path)
	if err != nil {
		return "", err
	}
	isSkipped, err := l.isSkipped(ctx, skipKey{vault: vault, path: path, size: stat.Size, modTime: stat.ModTime})
	if err != nil {
		return "", err
	}
	if isSkipped {
		return StateSkipped, nil
	}
	return l.startNew(ctx, vault, path)
}

func (l *Launcher) statFile(ctx context.Context, path string) (domain.VaultID, domain.FileStat, error) {
	vault, err := l.recognitions.GetVaultID(ctx, "")
	if err != nil {
		return "", domain.FileStat{}, err
	}
	stat, err := l.recognitions.StatFile(ctx, vault, path)
	if err != nil {
		return "", domain.FileStat{}, err
	}
	return vault, stat, nil
}

func (l *Launcher) startNew(ctx context.Context, vault domain.VaultID, path string) (string, error) {
	state, err := l.start(ctx, vault, path)
	if err != nil {
		return "", err
	}
	if state == StateRunning {
		return "", fmt.Errorf("%s: %w", path, recognition.ErrAlreadyRecognizing)
	}
	return state, nil
}

// isSkipped reports whether the mode leaves the file alone, and remembers it when so.
func (l *Launcher) isSkipped(ctx context.Context, key skipKey) (bool, error) {
	l.mu.Lock()
	mode := l.mode
	_, isKnown := l.skipped[key]
	l.mu.Unlock()
	if mode != settings.ModeWhenNoTextLayer {
		return false, nil
	}
	if isKnown {
		return true, nil
	}
	hasLayer, err := l.recognitions.HasTextLayer(ctx, key.vault, key.path)
	if err != nil || !hasLayer {
		return false, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.mode == mode {
		l.skipped[key] = struct{}{}
	}
	return true, nil
}

func (l *Launcher) start(ctx context.Context, vault domain.VaultID, path string) (string, error) {
	key := string(vault) + "\x00" + path
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.isClosed {
		return "", errors.New("the plugin is stopping")
	}
	runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	runEnd, isStarted, err := l.recognitions.Start(runCtx, recognition.Request{Vault: vault, Path: path})
	if err != nil || !isStarted {
		cancel()
		if err != nil {
			return "", err
		}
		return StateRunning, nil
	}
	l.running[key] = cancel
	l.wg.Go(func() {
		<-runEnd
		cancel()
		l.clearRunning(key)
	})
	return StateStarted, nil
}

func (l *Launcher) clearRunning(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.running, key)
}
