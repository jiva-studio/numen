package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/launch"
)

// The instruction sentences of recognition_create: when to call it and what it returns.
const (
	recognitionUseSentence        = "Use recognition_create on a scanned PDF or an image that has no text layer, or on a file whose search results are unreadable text."
	recognitionSlowSentence       = "Recognition is slow, up to an hour for a book."
	recognitionBackgroundSentence = "recognition_create returns a task id at once and the recognition runs in the background; once it ends, the recognized text can be read and searched."
	recognitionOnceSentence       = "Call recognition_create once per file: a second call while the first runs starts nothing and reports the state running, and a call on a file already recognized starts nothing and reports the state done with the version that is read."
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

func newToolRegistry(launcher *launch.Launcher) (*tools.Registry, error) {
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
	}, newCreateRecognition(launcher))
	if err != nil {
		return nil, err
	}
	return registry, nil
}

func newCreateRecognition(launcher *launch.Launcher) func(context.Context, RecognitionCreateInput) (RecognitionCreateOutput, error) {
	return func(ctx context.Context, in RecognitionCreateInput) (RecognitionCreateOutput, error) {
		out, err := launcher.Create(ctx, domain.VaultID(tools.GetVault(ctx)), in.Path)
		if err != nil {
			return RecognitionCreateOutput{}, err
		}
		return RecognitionCreateOutput{TaskID: out.TaskID, Path: out.Path, State: out.State, Version: out.Version}, nil
	}
}
