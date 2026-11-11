package engines

import (
	"context"
	"errors"
	"image"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/recognition"
)

type silentReporter struct{}

func (silentReporter) ReportCloseError(error) {}

func newHolder(open Opener) *Holder {
	return NewHolder(settings.Defaults(), open, silentReporter{})
}

// opened counts the recognizers a factory opened and the ones closed.
type opened struct {
	opened, closed int
}

func (o *opened) open(context.Context, settings.Config) (Recognizer, error) {
	o.opened++
	return closing{o: o}, nil
}

// fakeRecognizer reads no page.
type fakeRecognizer struct{}

func (fakeRecognizer) Close() error { return nil }

func (fakeRecognizer) GetRecognitionModel() recognition.Model { return recognition.Model{} }

func (fakeRecognizer) Recognize(context.Context, image.Image) ([]recognition.Block, error) {
	return nil, nil
}

type closing struct {
	fakeRecognizer
	o *opened
}

func (c closing) Close() error {
	c.o.closed++
	return nil
}

func TestUpdateConfigReopensEngine(t *testing.T) {
	o := &opened{}
	models := newHolder(o.open)
	first, err := models.GetEngine(t.Context())
	if err != nil || first == nil {
		t.Fatalf("first getEngine: %v, %v", first, err)
	}
	second, err := models.GetEngine(t.Context())
	if err != nil || second == nil || o.opened != 1 {
		t.Fatalf("cached getEngine: %v, %v, opened %d", second, err, o.opened)
	}

	models.UpdateSettings(map[string]string{"dpi": "1"})
	if models.cfg.Recognize.DPI != 1 || o.closed != 1 {
		t.Fatalf("config %+v, %d closed; want DPI 1 and the recognizer closed", models.cfg.Recognize, o.closed)
	}
	if eng, err := models.GetEngine(t.Context()); err != nil || eng == nil || o.opened != 2 {
		t.Fatalf("reopened after DPI change: %v, %v, opened %d", eng, err, o.opened)
	}

	models.UpdateSettings(map[string]string{"dpi": "450", "model": "custom", "customModelPath": "/opt/models/rec.onnx"})
	if models.cfg.Recognize.DPI != 450 || models.cfg.Recognize.Path != "/opt/models/rec.onnx" || o.closed != 2 {
		t.Fatalf("config %+v, %d closed; want the new settings and the recognizer closed", models.cfg.Recognize, o.closed)
	}
	for _, invalid := range []string{"-100", "0", "not-a-number"} {
		models.UpdateSettings(map[string]string{"dpi": invalid, "model": "custom", "customModelPath": "/opt/models/rec.onnx"})
		if models.cfg.Recognize.DPI != 450 {
			t.Errorf("a resolution of %q set it to %d", invalid, models.cfg.Recognize.DPI)
		}
	}
	if eng, err := models.GetEngine(t.Context()); err != nil || eng == nil || o.opened != 3 {
		t.Fatalf("%d opened (%v), want the recognizer opened again", o.opened, err)
	}
	models.UpdateSettings(map[string]string{"dpi": "450", "model": "custom", "customModelPath": "/opt/models/rec.onnx"})
	if o.closed != 2 {
		t.Fatalf("%d closed: an unchanged setting closed the recognizer", o.closed)
	}
	if err := models.Close(); err != nil || o.closed != 3 {
		t.Fatalf("%d closed (%v): close did not close the recognizer", o.closed, err)
	}
}

func TestGetEngineReturnsOpenError(t *testing.T) {
	models := newHolder(func(context.Context, settings.Config) (Recognizer, error) {
		return nil, errors.New("no models")
	})
	if _, err := models.GetEngine(t.Context()); err == nil {
		t.Fatal("a recognizer that could not open was opened")
	}
}

// The recognizer opens with the model the settings name, and the paths of a custom model only
// while the custom model is chosen.
func TestTheRecognizerOpensWithTheModelOfTheSettings(t *testing.T) {
	var got settings.Config
	models := newHolder(func(_ context.Context, cfg settings.Config) (Recognizer, error) {
		got = cfg
		return fakeRecognizer{}, nil
	})
	for _, tc := range []struct {
		values     map[string]string
		model      string
		path, dict string
	}{
		{map[string]string{"model": "pp-ocrv5-eslav"}, "eslav_PP-OCRv5_rec_mobile.onnx", "", ""},
		{map[string]string{"model": "custom", "customModelPath": "/m/rec.onnx", "customDictPath": "/m/dict.txt"}, settings.RecognizerCustom, "/m/rec.onnx", "/m/dict.txt"},
		{map[string]string{"model": "pp-ocrv6-tiny"}, "PP-OCRv6_rec_tiny.onnx", "", ""},
		{map[string]string{"model": "tesseract"}, "PP-OCRv6_rec_tiny.onnx", "", ""},
	} {
		models.UpdateSettings(tc.values)
		if _, err := models.GetEngine(t.Context()); err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(got.Recognize.Name, tc.model) || got.Recognize.Path != tc.path || got.Recognize.Dict != tc.dict {
			t.Errorf("settings %v: opened %+v", tc.values, got.Recognize)
		}
	}
}

func TestCustomModelFilesAreReadFromTheCamelCaseKeys(t *testing.T) {
	models := newHolder((&opened{}).open)
	models.UpdateSettings(map[string]string{
		"model": "custom", "customModelPath": "/m/rec.onnx", "customDictPath": "/m/dict.txt",
	})

	got := models.cfg.Recognize
	if got.Path != "/m/rec.onnx" || got.Dict != "/m/dict.txt" {
		t.Errorf("model path %q and dictionary %q, want /m/rec.onnx and /m/dict.txt", got.Path, got.Dict)
	}
}

func TestCustomModelFilesAreNotReadFromTheSnakeCaseKeys(t *testing.T) {
	models := newHolder((&opened{}).open)
	models.UpdateSettings(map[string]string{
		"model": "custom", "custom_model_path": "/m/rec.onnx", "custom_dict_path": "/m/dict.txt",
	})

	got := models.cfg.Recognize
	if got.Path != "" || got.Dict != "" {
		t.Errorf("model path %q and dictionary %q, want none", got.Path, got.Dict)
	}
}
