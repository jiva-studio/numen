package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/recognition"
)

// instructions is the paragraph that tells an agent when to call recognition_create.
const instructions = "Use recognition_create on a scanned PDF or an image that has no text layer, or on a file " +
	"whose search results are unreadable text. Recognition is slow, up to an hour for a book. " +
	"The call returns a task id at once and the recognition runs in the background; once it ends, " +
	"the recognised text can be read and searched. Call it once per file: a second call " +
	"while the first runs starts nothing and reports the state running."

// The states recognition_create reports.
const (
	stateStarted = "started"
	stateRunning = "running"
)

var errNoVault = errors.New("the call names no vault")

// RecognitionCreateInput is the input of the recognition_create tool.
type RecognitionCreateInput struct {
	Path string `json:"path" jsonschema:"the scanned PDF or image, relative to the vault root"`
}

// RecognitionCreateOutput is the output of the recognition_create tool.
type RecognitionCreateOutput struct {
	TaskID string `json:"task_id" jsonschema:"the id of the recognition in the list of tasks"`
	Path   string `json:"path" jsonschema:"the file; library_read and library_search read its text once it is recognised"`
	State  string `json:"state" jsonschema:"started, or running where a recognition of this file is already in progress and no second one was started"`
}

// Tools serves recognition_create, which starts recognitions that run until they end or Close.
type Tools struct {
	recognitions *recognition.Recognition

	wg sync.WaitGroup

	mu       sync.Mutex
	isClosed bool
	running  map[string]context.CancelFunc
}

// NewTools is the tools over the recognitions.
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
	registry.SetInstructions(instructions)
	err := tools.Add(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "recognition_create",
		Title: "Recognise a scanned document",
		Description: "Start the text recognition of one scanned PDF or image of the vault, with the model the " +
			"plugin's setting selects, and return its task id without waiting. The recognised text is stored " +
			"as a text version of the file, which library_read and library_search read once the recognition " +
			"ends. A recognition takes up to an hour for a book. A file that is not in the vault is refused.",
		Effect:  tools.ReadWrite,
		Servers: []tools.Server{tools.Full},
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
		return RecognitionCreateOutput{}, fmt.Errorf("find %s: %w", in.Path, err)
	}
	state, err := t.start(ctx, vault, in.Path)
	if err != nil {
		return RecognitionCreateOutput{}, err
	}
	return RecognitionCreateOutput{TaskID: recognition.GetTaskID(in.Path), Path: in.Path, State: state}, nil
}

// start runs the recognition of the file in the background, and reports running where another run
// holds the file.
func (t *Tools) start(ctx context.Context, vault domain.VaultID, path string) (string, error) {
	key := string(vault) + "\x00" + path
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.isClosed {
		return "", errors.New("the plugin is stopping")
	}
	runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	runEnd, isStarted, err := t.recognitions.Start(runCtx, recognition.Request{Vault: vault, Path: path, IsUserStarted: true})
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
