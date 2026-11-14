package launch_test

import (
	"context"
	"errors"
	"image"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/launch"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/recognition"
)

const (
	pluginID  = "md.numen.engines.ocr"
	bookPath  = "books/text.pdf"
	bookBytes = "%PDF text"
)

// countingDocuments opens every file as a document with the text layer it is set to, and counts the openings.
type countingDocuments struct {
	hasLayer bool
	failure  error
	opened   atomic.Int32
	asked    atomic.Int32
}

func (d *countingDocuments) Open(_ string, _ []byte) (recognition.Document, error) {
	d.opened.Add(1)
	if d.failure != nil {
		return nil, d.failure
	}
	return onePage{hasLayer: d.hasLayer, asked: &d.asked}, nil
}

type onePage struct {
	hasLayer bool
	asked    *atomic.Int32
}

func (onePage) GetPageCount() int    { return 1 }
func (p onePage) HasTextLayer() bool {
	p.asked.Add(1)
	return p.hasLayer
}
func (onePage) Close() error         { return nil }

func (onePage) RenderPage(_, _ int) (image.Image, error) {
	return image.NewRGBA(image.Rect(0, 0, 10, 10)), nil
}

type emptyRecognizer struct{}

func (emptyRecognizer) GetRecognitionModel() recognition.Model {
	return recognition.Model{Layout: "layout", Recognizer: "rec", DPI: 72, From: "test"}
}

func (emptyRecognizer) Recognize(_ context.Context, _ image.Image) ([]recognition.Block, error) {
	return nil, nil
}

type modeScene struct {
	vault     *hosttest.Vault
	queue     *hosttest.Queue
	documents *countingDocuments
	launcher  *launch.Launcher
}

func newModeScene(t *testing.T, hasLayer bool) *modeScene {
	t.Helper()
	vault := hosttest.New("Physics")
	vault.File(bookPath, bookBytes)
	texts := hosttest.NewTexts("Physics", pluginID)
	texts.SetStore(pluginID, vault.Store())
	texts.File(bookPath, []byte(bookBytes))
	s := &modeScene{vault: vault, queue: hosttest.NewQueue(), documents: &countingDocuments{hasLayer: hasLayer}}
	recognitions, err := recognition.New(
		recognition.Host{
			Vaults:    func(id domain.VaultID) recognition.Files { return vault.With(id) },
			Store:     func(id domain.VaultID) recognition.Folder { return vault.Store().With(id) },
			Texts:     func(id domain.VaultID) host.TextVersions { return texts.With(id) },
			Queue:     s.queue,
			Documents: s.documents,
		},
		func(context.Context) (recognition.Recognizer, error) { return emptyRecognizer{}, nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	s.launcher = launch.New(recognitions, pluginID)
	t.Cleanup(s.launcher.Close)
	return s
}

func (s *modeScene) startFile(t *testing.T) string {
	t.Helper()
	state, err := s.launcher.StartMissingFile(t.Context(), bookPath)
	if err != nil {
		t.Fatalf("StartMissingFile: %v", err)
	}
	return state
}

func TestAFileSkippedForItsTextIsNotAskedTwiceWhileItIsUnchanged(t *testing.T) {
	s := newModeScene(t, true)
	s.launcher.SetMode(settings.ModeWhenNoTextLayer)

	first, second := s.startFile(t), s.startFile(t)

	if first != launch.StateSkipped || second != launch.StateSkipped {
		t.Errorf("states %q and %q, want both skipped", first, second)
	}
	if opened := s.documents.opened.Load(); opened != 1 {
		t.Errorf("the document was opened %d times for two passes over an unchanged file, want 1", opened)
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("a skipped file shows the tasks %+v", tasks)
	}
}

func TestASkippedFileIsAskedAgainWhenItsModificationTimeChanges(t *testing.T) {
	s := newModeScene(t, true)
	s.launcher.SetMode(settings.ModeWhenNoTextLayer)
	s.startFile(t)

	s.vault.File(bookPath, bookBytes)
	state := s.startFile(t)

	if state != launch.StateSkipped {
		t.Errorf("the touched file with text gave %q, want skipped", state)
	}
	if opened := s.documents.opened.Load(); opened != 2 {
		t.Errorf("the document was opened %d times, want 2: once for each version of the file", opened)
	}
}

func TestASkippedFileIsAskedAgainWhenItsSizeChanges(t *testing.T) {
	s := newModeScene(t, true)
	s.launcher.SetMode(settings.ModeWhenNoTextLayer)
	s.startFile(t)

	s.vault.File(bookPath, bookBytes+" with a page more")
	s.startFile(t)

	if opened := s.documents.opened.Load(); opened != 2 {
		t.Errorf("the document was opened %d times, want 2: once for each size of the file", opened)
	}
}

func TestAChangeOfTheModeMakesTheLauncherForgetTheSkippedFiles(t *testing.T) {
	s := newModeScene(t, true)
	s.launcher.SetMode(settings.ModeWhenNoTextLayer)
	s.startFile(t)

	s.launcher.SetMode(settings.ModeOff)
	s.launcher.SetMode(settings.ModeWhenNoTextLayer)
	s.startFile(t)

	if opened := s.documents.opened.Load(); opened != 2 {
		t.Errorf("the document was opened %d times, want 2: the skip was kept across a change of the mode", opened)
	}
}

func TestAFileWithNoTextLayerIsStartedUnderWhenNoTextLayer(t *testing.T) {
	s := newModeScene(t, false)
	s.launcher.SetMode(settings.ModeWhenNoTextLayer)

	if state := s.startFile(t); state != launch.StateStarted {
		t.Errorf("a scan gave %q, want started", state)
	}
}

func TestAFileThatCannotBeOpenedToCheckItFailsTheRunWithTheError(t *testing.T) {
	s := newModeScene(t, false)
	s.documents.failure = errors.New("the file is damaged")
	s.launcher.SetMode(settings.ModeWhenNoTextLayer)

	_, err := s.launcher.StartMissingFile(t.Context(), bookPath)

	if !errors.Is(err, s.documents.failure) {
		t.Errorf("StartMissingFile returned %v, want the failure to open the file", err)
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("a file that failed the check shows the tasks %+v", tasks)
	}
}

func TestStartFileRecognizesAFileWithTextWithoutAskingForItsTextLayerWhateverTheMode(t *testing.T) {
	for _, mode := range []settings.RecognizeMode{settings.ModeOff, settings.ModeWhenNoTextLayer, settings.ModeAlways} {
		s := newModeScene(t, true)
		s.launcher.SetMode(mode)

		state, err := s.launcher.StartFile(t.Context(), bookPath)

		if err != nil || state != launch.StateStarted {
			t.Errorf("mode %s: StartFile gave %q, %v; want started", mode, state, err)
		}
		if asked := s.documents.asked.Load(); asked != 0 {
			t.Errorf("mode %s: StartFile asked for the text layer %d times, want never", mode, asked)
		}
	}
}

func TestStartMissingFileStartsAFileWithTextUnderAlwaysWithoutAskingForItsTextLayer(t *testing.T) {
	s := newModeScene(t, true)
	s.launcher.SetMode(settings.ModeAlways)

	state := s.startFile(t)

	if state != launch.StateStarted {
		t.Errorf("a file with text gave %q under always, want started", state)
	}
	if asked := s.documents.asked.Load(); asked != 0 {
		t.Errorf("the text layer was asked for %d times under always, want never", asked)
	}
}
