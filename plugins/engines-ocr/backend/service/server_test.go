package service

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
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

func TestASettingThatDecidesWhatReadsAPageOpensTheRecogniserAgain(t *testing.T) {
	o := &opened{}
	models := newEngines(engine.Defaults(), o.open)
	if _, err := models.getEngine(t.Context()); err != nil {
		t.Fatal(err)
	}

	models.updateConfig(map[string]string{"dpi": "450", "custom_model_path": "/opt/models/rec.onnx"})
	if models.cfg.Recognise.DPI != 450 || models.cfg.Recognise.Path != "/opt/models/rec.onnx" || o.closed != 1 {
		t.Fatalf("config %+v, %d closed; want the new settings and the recogniser closed", models.cfg.Recognise, o.closed)
	}
	for _, invalid := range []string{"-100", "0", "not-a-number"} {
		models.updateConfig(map[string]string{"dpi": invalid})
		if models.cfg.Recognise.DPI != 450 {
			t.Errorf("a resolution of %q set it to %d", invalid, models.cfg.Recognise.DPI)
		}
	}
	if _, err := models.getEngine(t.Context()); err != nil || o.opened != 2 {
		t.Fatalf("%d opened (%v), want the recogniser opened again", o.opened, err)
	}
	models.updateConfig(map[string]string{"language": models.cfg.Recognise.Name})
	if err := models.close(); err != nil || o.closed != 2 {
		t.Fatalf("%d closed (%v): an unchanged setting closed the recogniser, or close did not", o.closed, err)
	}
}

func TestTheHostRunningOnMockModelsOpensTheMock(t *testing.T) {
	t.Setenv("NUMEN_MOCK_ML", "true")
	opened, err := openEngine(t.Context(), engine.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if _, isMock := opened.(engine.MockRecogniser); !isMock {
		t.Fatalf("opened %T, want the mock", opened)
	}
}

func TestTheRecogniseCommandSaysWhatItRead(t *testing.T) {
	v := hosttest.New("Physics")
	raw, err := os.ReadFile("../document/testdata/scan.pdf")
	if err != nil {
		t.Fatal(err)
	}
	v.File("books/scan.pdf", string(raw))
	recognitions, err := recognition.New(
		recognition.Host{Vault: v, Store: v.Store(), Texts: hosttest.NewTexts("Physics", "md.numen.engines.ocr"), Queue: hosttest.NewQueue()},
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

func TestARecogniserThatCannotOpenIsAnError(t *testing.T) {
	models := newEngines(engine.Defaults(), func(context.Context, engine.Config) (recogniser, error) {
		return nil, errors.New("no models")
	})
	if _, err := models.getEngine(t.Context()); err == nil {
		t.Fatal("a recogniser that could not open was opened")
	}
}
