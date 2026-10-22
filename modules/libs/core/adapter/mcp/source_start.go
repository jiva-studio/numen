package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

func addSourceWriteTools(server *sdk.Server, core Core) {
	addSourceRecognise(server, core)
	addSourceTranscribe(server, core)
}

type sourceStartInput struct {
	Path string `json:"path" jsonschema:"the document, as source_list gives it"`
}

type sourceTranscribeInput struct {
	Path string `json:"path" jsonschema:"the recording, as a path inside the vault"`
}

type sourceStartOutput struct {
	IsStarted bool   `json:"started" jsonschema:"whether the vault took this on, which it always does"`
	Status    string `json:"status" jsonschema:"what is happening now, in words to say back to the person"`
}

func addSourceRecognise(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "source_recognise",
		Title: "Recognise a scanned document",
		Description: "Have a model read one scanned document and write what it says into " +
			"the vault. The words land as files under `.numen/ocr/`, beside the person's " +
			"notes and inside the folder they sync — a book is megabytes of them. The " +
			"pages it has read are searchable as it goes, so a search finds the " +
			"beginning of a book long before the end of it is read; the part not yet " +
			"read answers nothing until it is. This is slow — an hour for a book — and " +
			"it is never done on its own, because whether a document's own text is any " +
			"good cannot be told from the text. Ask for it when a document is a scan, " +
			"or when what a search returns from one is nonsense. A document asked for " +
			"while another is being read waits its turn and is never rejected, so this " +
			"is asked once and no more.",
	}, newHandler(func(_ context.Context, in sourceStartInput) (sourceStartOutput, error) {
		//nolint:contextcheck // a recognition runs under the application and goes on after the request that began it
		return startRecognition(core, in)
	}))
}

// startRecognition queues scan recognition and returns without waiting for it to finish.
func startRecognition(core Core, in sourceStartInput) (sourceStartOutput, error) {
	if core.Sources.StartRecognition == nil || core.Sources.GetRecognitionStatus == nil {
		return sourceStartOutput{}, fmt.Errorf("this installation cannot read scans")
	}
	status := "started; it runs in the background, the pages it has read are searchable " +
		"as it goes, and the window shows how far it has got"
	started := core.Sources.StartRecognition.Execute(core.getCurrentVault().Vault, in.Path)
	switch {
	case started == source.Queued:
		status = "queued; another document is being read and this one is in line behind " +
			"it — nothing more is needed, it begins when that recognition is over"
	case !core.Sources.GetRecognitionStatus.Execute().IsReady:
		status = "started; what is needed to read scans is being fetched first, about 160 MB"
	}
	return sourceStartOutput{IsStarted: true, Status: status}, nil
}

func addSourceTranscribe(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "source_transcribe",
		Title: "Transcribe a recording",
		Description: "Have a model listen to one recording and write the words it carries " +
			"into the vault. They land as files under `.numen/transcript/`, beside the person's " +
			"notes and inside the folder they sync — an hour of talk is megabytes of " +
			"them. What has been transcribed is searchable as it goes, so a search " +
			"finds the first minutes of a talk long before the last of them are " +
			"reached. This is slow — about as long as the recording itself. A vault's " +
			"recordings are transcribed on their own where the installation is set to; " +
			"ask for this when one is wanted now, or when the installation leaves it to " +
			"the hand. A recording asked for by name is transcribed whatever the " +
			"installation does on its own, and waits behind nothing but the recordings " +
			"asked for before it, so this is asked once and no more.",
	}, newHandler(func(_ context.Context, in sourceTranscribeInput) (sourceStartOutput, error) {
		//nolint:contextcheck // a transcription runs under the application and goes on after the request that began it
		return startTranscription(core, in)
	}))
}

// startTranscription queues transcription and returns without waiting for it to finish.
func startTranscription(core Core, in sourceTranscribeInput) (sourceStartOutput, error) {
	if core.Sources.StartTranscription == nil || core.Sources.GetTranscriptionReadiness == nil {
		return sourceStartOutput{}, fmt.Errorf("this installation cannot hear recordings")
	}
	status := "started; it runs in the background, and what has been transcribed is " +
		"searchable as it goes"
	started := core.Sources.StartTranscription.Execute(core.getCurrentVault().Vault, in.Path)
	switch {
	case started == source.Queued:
		status = "queued; another recording is being transcribed and this one is in line " +
			"behind it — nothing more is needed, it begins when that one is over"
	case !core.Sources.GetTranscriptionReadiness.Execute():
		status = "started; what is needed to transcribe recordings is being fetched first"
	}
	return sourceStartOutput{IsStarted: true, Status: status}, nil
}
