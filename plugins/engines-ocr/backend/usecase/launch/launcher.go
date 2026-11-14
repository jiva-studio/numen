// Package launch starts the recognitions of files and keeps them running until they end or the
// launcher is closed.
package launch

import (
	"context"
	"errors"
	"fmt"
	"sync"

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

// SetMode sets when StartFile recognizes a file.
func (l *Launcher) SetMode(_ settings.RecognizeMode) {}

// StartFile starts the recognition of a file of the shown vault and fails when one already runs. It
// returns StateSkipped for a file the mode leaves alone, and StateStarted otherwise.
func (l *Launcher) StartFile(ctx context.Context, path string) (string, error) {
	vault, err := l.recognitions.GetVaultID(ctx, "")
	if err != nil {
		return "", err
	}
	if _, err := l.recognitions.StatFile(ctx, vault, path); err != nil {
		return "", err
	}
	state, err := l.start(ctx, vault, path)
	if err != nil {
		return "", err
	}
	if state == StateRunning {
		return "", fmt.Errorf("%s: %w", path, recognition.ErrAlreadyRecognizing)
	}
	return state, nil
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
