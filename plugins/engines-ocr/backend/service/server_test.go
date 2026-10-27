package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

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
	if _, err := models.getEngine(t.Context()); err != nil {
		t.Fatal(err)
	}

	models.updateConfig(map[string]string{"dpi": "450", "model": "custom", "custom_model_path": "/opt/models/rec.onnx"})
	if models.cfg.Recognise.DPI != 450 || models.cfg.Recognise.Path != "/opt/models/rec.onnx" || o.closed != 1 {
		t.Fatalf("config %+v, %d closed; want the new settings and the recogniser closed", models.cfg.Recognise, o.closed)
	}
	for _, invalid := range []string{"-100", "0", "not-a-number"} {
		models.updateConfig(map[string]string{"dpi": invalid, "model": "custom", "custom_model_path": "/opt/models/rec.onnx"})
		if models.cfg.Recognise.DPI != 450 {
			t.Errorf("a resolution of %q set it to %d", invalid, models.cfg.Recognise.DPI)
		}
	}
	if _, err := models.getEngine(t.Context()); err != nil || o.opened != 2 {
		t.Fatalf("%d opened (%v), want the recogniser opened again", o.opened, err)
	}
	models.updateConfig(map[string]string{"dpi": "450", "model": "custom", "custom_model_path": "/opt/models/rec.onnx"})
	if err := models.close(); err != nil || o.closed != 2 {
		t.Fatalf("%d closed (%v): an unchanged setting closed the recogniser, or close did not", o.closed, err)
	}
}

func TestOpenEngineReturnsMockRecogniser(t *testing.T) {
	t.Setenv(EnvMockML, "true")
	opened, err := openEngine(t.Context(), engine.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if _, isMock := opened.(engine.MockRecogniser); !isMock {
		t.Fatalf("opened %T, want the mock", opened)
	}
}

func TestRecogniseCommandReportsPages(t *testing.T) {
	v := hosttest.New("Physics")
	raw, err := os.ReadFile("../document/testdata/scan.pdf")
	if err != nil {
		t.Fatal(err)
	}
	v.File("books/scan.pdf", string(raw))
	texts := hosttest.NewTexts("Physics", "md.numen.engines.ocr")
	texts.SetStore("md.numen.engines.ocr", v.Store())
	recognitions, err := recognition.New(
		recognition.Host{Vault: v, Store: v.Store(), Texts: texts, Queue: hosttest.NewQueue()},
		func(context.Context) (recognition.Recogniser, error) { return engine.MockRecogniser{}, nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	run := newRecogniseCommand(recognitions)

	res, err := run(t.Context(), command.Request{Command: "recognise", Path: "books/scan.pdf"})
	if err != nil || !strings.Contains(res.Message, "recognised 2 pages of books/scan.pdf") {
		t.Fatalf("answered %+v (%v)", res, err)
	}
	if _, err := run(t.Context(), command.Request{Command: "recognise", Path: "books/gone.pdf"}); err == nil {
		t.Fatal("a file that is not there was recognised")
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
		{map[string]string{"model": "custom", "custom_model_path": "/m/rec.onnx", "custom_dict_path": "/m/dict.txt"}, engine.RecogniserCustom, "/m/rec.onnx", "/m/dict.txt"},
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

// Each model the manifest offers is one the engine runs, fetched from the address the engine knows.
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
