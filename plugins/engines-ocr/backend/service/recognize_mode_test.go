package service

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/document"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/launch"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/recognition"
)

const (
	// textPath is a PDF of the shown vault that carries text of its own.
	textPath = "books/text.pdf"
	// picturePath is an image of the shown vault.
	picturePath = "books/page.png"
)

// layerSpy counts the questions whether a document has text of its own.
type layerSpy struct {
	recognition.Document
	asked *atomic.Int32
}

func (d layerSpy) HasTextLayer() bool {
	d.asked.Add(1)
	return d.Document.HasTextLayer()
}

// spyDocuments opens documents as the plugin does and counts the questions asked of them.
type spyDocuments struct {
	inner recognition.Documents
	asked atomic.Int32
}

func (d *spyDocuments) Open(filePath string, raw []byte) (recognition.Document, error) {
	doc, err := d.inner.Open(filePath, raw)
	if err != nil {
		return nil, err
	}
	return layerSpy{Document: doc, asked: &d.asked}, nil
}

func newModeFixture(t *testing.T, mode settings.RecognizeMode) (*toolFixture, *spyDocuments) {
	t.Helper()
	s := newToolFixture(t)
	close(s.recognizer.release)
	layered, err := os.ReadFile("../internal/adapter/document/testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	var picture bytes.Buffer
	if err := png.Encode(&picture, image.NewRGBA(image.Rect(0, 0, 10, 10))); err != nil {
		t.Fatal(err)
	}
	for filePath, raw := range map[string][]byte{textPath: layered, picturePath: picture.Bytes()} {
		s.vault.File(filePath, string(raw))
		s.texts.File(filePath, raw)
	}
	documents := &spyDocuments{inner: document.NewReader()}
	recognitions, err := recognition.New(
		recognition.Host{
			Vaults:    func(id domain.VaultID) recognition.Files { return s.vault.With(id) },
			Store:     foldersOf(s.vault.Store()),
			Texts:     textsOf(s.texts),
			Queue:     s.queue,
			Documents: documents,
		},
		func(context.Context) (recognition.Recognizer, error) { return s.recognizer, nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	s.recognitions = recognitions
	s.launcher = launch.New(recognitions, PluginID)
	t.Cleanup(s.launcher.Close)
	s.launcher.SetMode(mode)
	s.registry, err = newToolRegistry(s.launcher)
	if err != nil {
		t.Fatal(err)
	}
	return s, documents
}

// missingTextCommand is the shared command the trigger of the manifest runs.
const missingTextCommand = "app:file.createMissingText"

func (s *toolFixture) runCommand(t *testing.T, id, filePath string) command.Result {
	t.Helper()
	res, err := s.runHandlerWithin(t, command.Request{Command: id, Path: filePath})
	if err != nil {
		t.Fatalf("%s of %s: %v", id, filePath, err)
	}
	return res
}

func (s *toolFixture) createText(t *testing.T, filePath string) command.Result {
	t.Helper()
	return s.runCommand(t, "app:file.createText", filePath)
}

func (s *toolFixture) createMissingText(t *testing.T, filePath string) command.Result {
	t.Helper()
	return s.runCommand(t, missingTextCommand, filePath)
}

func (s *toolFixture) expectRecognized(t *testing.T, filePath string) {
	t.Helper()
	s.waitUntilFinished(t, recognition.GetTaskID("Physics", filePath))
	if !hasRecognition(t, s.texts, filePath) {
		t.Errorf("%s has no recognition once its task ended", filePath)
	}
}

func (s *toolFixture) expectSkipped(t *testing.T, filePath string, res command.Result) {
	t.Helper()
	if !strings.Contains(res.Message, filePath) || !strings.Contains(res.Message, "skipped") {
		t.Errorf("the message %q, want it to say that %s is skipped", res.Message, filePath)
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("a skipped file shows the tasks %+v", tasks)
	}
	if hasRecognition(t, s.texts, filePath) {
		t.Errorf("%s was recognized although it was skipped", filePath)
	}
	select {
	case <-s.recognizer.startCh:
		t.Errorf("a page of the skipped %s was read", filePath)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestMissingTextWithModeWhenNoTextLayerSkipsAPDFThatHasText(t *testing.T) {
	s, _ := newModeFixture(t, settings.ModeWhenNoTextLayer)

	res := s.createMissingText(t, textPath)

	s.expectSkipped(t, textPath, res)
}

func TestMissingTextWithModeWhenNoTextLayerRecognizesAScannedPDF(t *testing.T) {
	s, _ := newModeFixture(t, settings.ModeWhenNoTextLayer)

	res := s.createMissingText(t, scanPath)

	if res.Message != "recognizing "+scanPath {
		t.Errorf("the message %q, want recognizing %s", res.Message, scanPath)
	}
	s.expectRecognized(t, scanPath)
}

func TestMissingTextWithModeWhenNoTextLayerRecognizesAnImage(t *testing.T) {
	s, _ := newModeFixture(t, settings.ModeWhenNoTextLayer)

	res := s.createMissingText(t, picturePath)

	if res.Message != "recognizing "+picturePath {
		t.Errorf("the message %q, want recognizing %s", res.Message, picturePath)
	}
	s.expectRecognized(t, picturePath)
}

func TestCreateTextWithModeWhenNoTextLayerRecognizesAPDFThatHasTextWithoutAskingForItsTextLayer(t *testing.T) {
	s, documents := newModeFixture(t, settings.ModeWhenNoTextLayer)

	res := s.createText(t, textPath)

	if res.Message != "recognizing "+textPath {
		t.Errorf("the message %q, want recognizing %s", res.Message, textPath)
	}
	s.expectRecognized(t, textPath)
	if asked := documents.asked.Load(); asked != 0 {
		t.Errorf("the text layer was asked for %d times by the menu command, want never", asked)
	}
}

func TestBothCommandsWithModeAlwaysRecognizeAPDFThatHasTextAndAScanWithoutAskingForTheTextLayer(t *testing.T) {
	s, documents := newModeFixture(t, settings.ModeAlways)

	s.createMissingText(t, textPath)
	s.expectRecognized(t, textPath)
	s.createMissingText(t, scanPath)
	s.expectRecognized(t, scanPath)
	s.createText(t, picturePath)
	s.expectRecognized(t, picturePath)

	if asked := documents.asked.Load(); asked != 0 {
		t.Errorf("the text layer was asked for %d times under always, want never", asked)
	}
}

func TestCreateTextWithModeOffRecognizesAPDFThatHasText(t *testing.T) {
	s, documents := newModeFixture(t, settings.ModeOff)

	res := s.createText(t, textPath)

	if res.Message != "recognizing "+textPath {
		t.Errorf("the message %q, want recognizing %s", res.Message, textPath)
	}
	s.expectRecognized(t, textPath)
	if asked := documents.asked.Load(); asked != 0 {
		t.Errorf("the text layer was asked for %d times under off, want never", asked)
	}
}

func TestTheToolRecognitionCreateRecognizesAPDFThatHasTextWithModeWhenNoTextLayer(t *testing.T) {
	s, _ := newModeFixture(t, settings.ModeWhenNoTextLayer)

	out := s.recognize(t, "Physics", textPath)

	if out.State != launch.StateStarted {
		t.Errorf("recognition_create returned the state %q, want started whatever the mode", out.State)
	}
	s.waitUntilFinished(t, out.TaskID)
}
