package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	ocr "github.com/jiva-studio/numen/plugins/engines-ocr"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/recognition"
)

// opened counts the recognisers a factory opened and the ones closed.
type opened struct {
	opened, closed int
}

func (o *opened) open(context.Context, engine.Config) (recogniser, error) {
	o.opened++
	return closing{o: o}, nil
}

type closing struct {
	engine.MockRecogniser
	o *opened
}

func (c closing) Close() error {
	c.o.closed++
	return nil
}

func TestUpdateConfigReopensEngine(t *testing.T) {
	o := &opened{}
	models := newEngines(engine.Defaults(), o.open)
	first, err := models.getEngine(t.Context())
	if err != nil || first == nil {
		t.Fatalf("first getEngine: %v, %v", first, err)
	}
	second, err := models.getEngine(t.Context())
	if err != nil || second == nil || o.opened != 1 {
		t.Fatalf("cached getEngine: %v, %v, opened %d", second, err, o.opened)
	}

	models.updateConfig(map[string]string{"dpi": "1"})
	if models.cfg.Recognise.DPI != 1 || o.closed != 1 {
		t.Fatalf("config %+v, %d closed; want DPI 1 and the recogniser closed", models.cfg.Recognise, o.closed)
	}
	if eng, err := models.getEngine(t.Context()); err != nil || eng == nil || o.opened != 2 {
		t.Fatalf("reopened after DPI change: %v, %v, opened %d", eng, err, o.opened)
	}

	models.updateConfig(map[string]string{"dpi": "450", "model": "custom", "customModelPath": "/opt/models/rec.onnx"})
	if models.cfg.Recognise.DPI != 450 || models.cfg.Recognise.Path != "/opt/models/rec.onnx" || o.closed != 2 {
		t.Fatalf("config %+v, %d closed; want the new settings and the recogniser closed", models.cfg.Recognise, o.closed)
	}
	for _, invalid := range []string{"-100", "0", "not-a-number"} {
		models.updateConfig(map[string]string{"dpi": invalid, "model": "custom", "customModelPath": "/opt/models/rec.onnx"})
		if models.cfg.Recognise.DPI != 450 {
			t.Errorf("a resolution of %q set it to %d", invalid, models.cfg.Recognise.DPI)
		}
	}
	if eng, err := models.getEngine(t.Context()); err != nil || eng == nil || o.opened != 3 {
		t.Fatalf("%d opened (%v), want the recogniser opened again", o.opened, err)
	}
	models.updateConfig(map[string]string{"dpi": "450", "model": "custom", "customModelPath": "/opt/models/rec.onnx"})
	if o.closed != 2 {
		t.Fatalf("%d closed: an unchanged setting closed the recogniser", o.closed)
	}
	if err := models.close(); err != nil || o.closed != 3 {
		t.Fatalf("%d closed (%v): close did not close the recogniser", o.closed, err)
	}
}

func TestOpenEngineReturnsMockRecogniser(t *testing.T) {
	t.Setenv(EnvMockML, "true")
	opened, err := newEngineOpener(engine.NewLibrary())(t.Context(), engine.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if _, isMock := opened.(engine.MockRecogniser); !isMock {
		t.Fatalf("opened %T, want the mock", opened)
	}
}

func (s *toolFixture) runHandlerWithin(t *testing.T, req command.Request) (command.Result, error) {
	t.Helper()
	commands, err := newCommands(s.tools)
	if err != nil {
		t.Fatal(err)
	}
	type answer struct {
		res command.Result
		err error
	}
	answers := make(chan answer, 1)
	go func() {
		res, err := commands.RunCommand(t.Context(), req)
		answers <- answer{res, err}
	}()
	select {
	case got := <-answers:
		return got.res, got.err
	case <-time.After(workDeadline):
		t.Fatalf("%s did not answer within %v while the file was recognised", req.Command, workDeadline)
		return command.Result{}, nil
	}
}

func TestTheHandlerOfCreateTextAnswersWhileTheTaskRecognises(t *testing.T) {
	s := newToolFixture(t)

	res, err := s.runHandlerWithin(t, command.Request{Command: "app:file.createText", Path: scanPath})

	if err != nil || res.Message != "recognising "+scanPath {
		t.Fatalf("answered %+v (%v)", res, err)
	}
	s.waitStarted(t)
	close(s.recogniser.release)
	s.waitFinished(t, recognition.GetTaskID(scanPath))
	if !hasRecognition(t, s.texts, scanPath) {
		t.Error("the scan has no recognition once its task ended")
	}
}

func TestCreateTextFailsTheTaskOfAFailedRecognition(t *testing.T) {
	s := newToolFixture(t)
	s.recogniser.failure = errors.New("the page cannot be decoded")
	close(s.recogniser.release)

	_, err := s.runHandlerWithin(t, command.Request{Command: "app:file.createText", Path: scanPath})

	if err != nil {
		t.Errorf("a failed recognition was answered %v, want the failure on its task alone", err)
	}
	deadline := time.After(workDeadline)
	for !hasFailedTask(s.queue.Queue, recognition.GetTaskID(scanPath), "the page cannot be decoded") {
		select {
		case <-deadline:
			t.Fatalf("no task failed within %v; tasks %+v", workDeadline, s.queue.ListTasks())
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func hasFailedTask(queue *hosttest.Queue, id, text string) bool {
	for _, one := range queue.ListTasks() {
		if one.ID == id && strings.Contains(one.FailureReason, text) {
			return true
		}
	}
	return false
}

func TestTheHandlerOfCreateTextFailsBeforeAnyTask(t *testing.T) {
	s := newToolFixture(t)

	if _, err := s.runHandlerWithin(t, command.Request{Command: "app:file.createText", Path: "books/gone.pdf"}); err == nil {
		t.Error("a file that is not there was answered as started")
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("the tasks %+v are shown for a file that is not there", tasks)
	}
	if _, err := s.runHandlerWithin(t, command.Request{Command: "app:version.delete", Path: scanPath}); !errors.Is(err, command.ErrUnknownCommand) {
		t.Errorf("Delete text, which the core runs, answered %v, want ErrUnknownCommand", err)
	}
	if _, err := s.runHandlerWithin(t, command.Request{Command: "recognition.create", Path: scanPath}); !errors.Is(err, command.ErrUnknownCommand) {
		t.Errorf("the command of before answered %v, want ErrUnknownCommand", err)
	}
}

func TestGetEngineReturnsOpenError(t *testing.T) {
	models := newEngines(engine.Defaults(), func(context.Context, engine.Config) (recogniser, error) {
		return nil, errors.New("no models")
	})
	if _, err := models.getEngine(t.Context()); err == nil {
		t.Fatal("a recogniser that could not open was opened")
	}
}

// The recogniser opens with the model the settings name, and the paths of a custom model only
// while the custom model is chosen.
func TestTheRecogniserOpensWithTheModelOfTheSettings(t *testing.T) {
	var got engine.Config
	models := newEngines(engine.Defaults(), func(_ context.Context, cfg engine.Config) (recogniser, error) {
		got = cfg
		return engine.MockRecogniser{}, nil
	})
	for _, tc := range []struct {
		values     map[string]string
		model      string
		path, dict string
	}{
		{map[string]string{"model": "pp-ocrv5-eslav"}, "eslav_PP-OCRv5_rec_mobile.onnx", "", ""},
		{map[string]string{"model": "custom", "customModelPath": "/m/rec.onnx", "customDictPath": "/m/dict.txt"}, engine.RecogniserCustom, "/m/rec.onnx", "/m/dict.txt"},
		{map[string]string{"model": "pp-ocrv6-tiny"}, "PP-OCRv6_rec_tiny.onnx", "", ""},
		{map[string]string{"model": "tesseract"}, "PP-OCRv6_rec_tiny.onnx", "", ""},
	} {
		models.updateConfig(tc.values)
		if _, err := models.getEngine(t.Context()); err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(got.Recognise.Name, tc.model) || got.Recognise.Path != tc.path || got.Recognise.Dict != tc.dict {
			t.Errorf("settings %v: opened %+v", tc.values, got.Recognise)
		}
	}
}

// Each model the manifest offers is one the engine runs, fetched from the engine's own address.
func TestEveryModelOfTheManifestIsOneTheEngineRuns(t *testing.T) {
	var manifest struct {
		Dependencies []struct {
			ID        string `json:"id"`
			SourceURL string `json:"sourceUrl"`
		} `json:"dependencies"`
		Settings []struct {
			Key     string `json:"key"`
			Options []struct {
				Value        string   `json:"value"`
				Dependencies []string `json:"dependencies"`
			} `json:"options"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(ocr.Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	sources := map[string]string{}
	for _, one := range manifest.Dependencies {
		sources[one.ID] = one.SourceURL
	}
	count := 0
	for _, setting := range manifest.Settings {
		if setting.Key != "model" {
			continue
		}
		for _, option := range setting.Options {
			count++
			address, ok := engine.FindRecogniser(option.Value)
			if !ok {
				t.Errorf("the manifest offers %s, which the engine does not run", option.Value)
				continue
			}
			var provisioned []string
			for _, id := range option.Dependencies {
				provisioned = append(provisioned, sources[id])
			}
			if address != "" && (len(provisioned) != 1 || provisioned[0] != address) {
				t.Errorf("%s provisions %v, and the engine fetches %s", option.Value, provisioned, address)
			}
		}
	}
	if count == 0 {
		t.Fatal("the manifest offers no model")
	}
}
