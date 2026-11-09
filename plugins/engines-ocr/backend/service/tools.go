package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/recognition"
)

// The instruction sentences of recognition_create: when to call it and what it returns.
const (
	recognitionUseSentence        = "Use recognition_create on a scanned PDF or an image that has no text layer, or on a file whose search results are unreadable text."
	recognitionSlowSentence       = "Recognition is slow, up to an hour for a book."
	recognitionBackgroundSentence = "recognition_create returns a task id at once and the recognition runs in the background; once it ends, the recognized text can be read and searched."
	recognitionOnceSentence       = "Call recognition_create once per file: a second call while the first runs starts nothing and reports the state running, and a call on a file already recognized starts nothing and reports the state done with the version that is read."
)

// The states recognition_create reports.
const (
	stateStarted = "started"
	stateRunning = "running"
	stateDone    = "done"
)

var (
	errNoVault  = errors.New("the call names no vault")
	errNotFound = errors.New("the file is not found")
)

// RecognitionCreateInput is the input of the recognition_create tool.
type RecognitionCreateInput struct {
	Path string `json:"path" jsonschema:"the scanned PDF or image, relative to the vault root"`
}

// RecognitionCreateOutput is the output of the recognition_create tool.
type RecognitionCreateOutput struct {
	TaskID  string `json:"taskId" jsonschema:"the id of the recognition in the list of tasks"`
	Path    string `json:"path" jsonschema:"the file; its text can be read and searched once it is recognized"`
	State   string `json:"state" jsonschema:"started; running when a recognition of this file is already in progress and no second one was started; or done when the file is already recognized and nothing was started"`
	Version string `json:"version,omitempty" jsonschema:"with the state done, the id of the live text version, the version that is read: the recognized text, or a version made from it such as a proofread"`
}

// Tools serves recognition_create, which starts recognitions that run until they end or Close.
type Tools struct {
	recognitions *recognition.Recognition

	wg sync.WaitGroup

	mu       sync.Mutex
	isClosed bool
	running  map[string]context.CancelFunc
}

// NewTools returns the tools for the recognitions.
func NewTools(recognitions *recognition.Recognition) *Tools {
	return &Tools{
		recognitions: recognitions,
		wg:           sync.WaitGroup{},
		mu:           sync.Mutex{},
		isClosed:     false,
		running:      map[string]context.CancelFunc{},
	}
}

// Registry registers and returns the recognition tool.
func (t *Tools) Registry() (*tools.Registry, error) {
	registry := tools.New()
	err := tools.Add(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "recognition_create",
		Title: "Recognize a scanned document",
		Description: "Start the text recognition of one scanned PDF or image of the vault, with the model the " +
			"plugin's setting selects, and return its task id without waiting. The recognized text is stored " +
			"as the text version of the file, which is read and searched once the recognition " +
			"ends. A recognition takes up to an hour for a book. A file that is not in the vault is rejected. A " +
			"file that is already recognized is not recognized again: the call returns the state done and the " +
			"version that is read, the recognized text or a version made from it.",
		Effect:       tools.ReadWrite,
		Servers:      []tools.Server{tools.Full},
		Instructions: []string{recognitionUseSentence, recognitionSlowSentence, recognitionBackgroundSentence, recognitionOnceSentence},
	}, t.createRecognition)
	if err != nil {
		return nil, err
	}
	return registry, nil
}

// Close cancels the recognitions the tool started and waits for them to end.
func (t *Tools) Close() {
	t.mu.Lock()
	t.isClosed = true
	for _, cancel := range t.running {
		cancel()
	}
	t.mu.Unlock()
	t.wg.Wait()
}

func (t *Tools) createRecognition(ctx context.Context, in RecognitionCreateInput) (RecognitionCreateOutput, error) {
	vault := domain.VaultID(tools.GetVault(ctx))
	if vault == "" {
		return RecognitionCreateOutput{}, errNoVault
	}
	if in.Path == "" {
		return RecognitionCreateOutput{}, errors.New("path required")
	}
	if _, err := t.recognitions.StatFile(ctx, vault, in.Path); err != nil {
		if errors.Is(err, host.ErrNotExist) {
			return RecognitionCreateOutput{}, fmt.Errorf("%s: %w", in.Path, errNotFound)
		}
		return RecognitionCreateOutput{}, fmt.Errorf("find %s: %w", in.Path, err)
	}
	out := RecognitionCreateOutput{TaskID: recognition.GetTaskID(vault, in.Path), Path: in.Path, State: stateRunning, Version: ""}
	isRunning, err := t.recognitions.IsRunning(ctx, vault, in.Path)
	if err != nil {
		return RecognitionCreateOutput{}, err
	}
	if isRunning {
		return out, nil
	}
	version, isFound, err := t.recognitions.FindLiveDerivedVersion(ctx, vault, in.Path, PluginID)
	if err != nil {
		return RecognitionCreateOutput{}, err
	}
	if isFound {
		out.State, out.Version = stateDone, string(version.GetID())
		return out, nil
	}
	out.State, err = t.start(ctx, vault, in.Path)
	if err != nil {
		return RecognitionCreateOutput{}, err
	}
	return out, nil
}

func (t *Tools) start(ctx context.Context, vault domain.VaultID, path string) (string, error) {
	key := string(vault) + "\x00" + path
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.isClosed {
		return "", errors.New("the plugin is stopping")
	}
	runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	runEnd, isStarted, err := t.recognitions.Start(runCtx, recognition.Request{Vault: vault, Path: path})
	if err != nil || !isStarted {
		cancel()
		if err != nil {
			return "", err
		}
		return stateRunning, nil
	}
	t.running[key] = cancel
	t.wg.Go(func() {
		<-runEnd
		cancel()
		t.clearRunning(key)
	})
	return stateStarted, nil
}

func (t *Tools) clearRunning(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.running, key)
}
