package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/wasm"
	ocr "github.com/jiva-studio/numen/plugins/engines-ocr"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/recognition"
)

type emptyRecogniser struct{}

func (emptyRecogniser) GetRecognitionModel() recognition.Model {
	return recognition.Model{Layout: "mock", Recogniser: "mock", DPI: 72, From: "mock"}
}

func (emptyRecogniser) Recognise(context.Context, image.Image) ([]recognition.Block, error) {
	return nil, nil
}

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
	if dir := os.Getenv("NUMEN_TEST_WASM_CACHE"); dir != "" {
		compilation := wasm.NewCompilationCache(dir)
		t.Cleanup(func() {
			if err := compilation.Close(context.WithoutCancel(t.Context())); err != nil {
				t.Error(err)
			}
		})
		recognitions.SetCompilationCache(compilation)
	}
	run := newRecogniseCommand(recognitions)

	res, err := run(t.Context(), command.Request{Command: "recognition.create", Path: "books/scan.pdf"})
	if err != nil || !strings.Contains(res.Message, "recognised 2 pages of books/scan.pdf") {
		t.Fatalf("answered %+v (%v)", res, err)
	}
	if _, err := run(t.Context(), command.Request{Command: "recognition.create", Path: "books/gone.pdf"}); err == nil {
		t.Fatal("a file that is not there was recognised")
	}

	emptyRecognitions, err := recognition.New(
		recognition.Host{Vault: v, Store: v.Store(), Texts: texts, Queue: hosttest.NewQueue()},
		func(context.Context) (recognition.Recogniser, error) { return emptyRecogniser{}, nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if dir := os.Getenv("NUMEN_TEST_WASM_CACHE"); dir != "" {
		compilation := wasm.NewCompilationCache(dir)
		t.Cleanup(func() {
			if err := compilation.Close(context.WithoutCancel(t.Context())); err != nil {
				t.Error(err)
			}
		})
		emptyRecognitions.SetCompilationCache(compilation)
	}
	emptyRes, err := newRecogniseCommand(emptyRecognitions)(t.Context(), command.Request{Command: "recognition.create", Path: "books/scan.pdf"})
	if err != nil || emptyRes.Message != "books/scan.pdf says nothing that could be read, and nothing was kept" {
		t.Fatalf("empty scan answered %+v (%v)", emptyRes, err)
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
		{map[string]string{"model": "pp-ocrv5-eslav"}, "rec-pp-ocrv5-eslav", "", ""},
		{map[string]string{"model": "custom", "customModelPath": "/m/rec.onnx", "customDictPath": "/m/dict.txt"}, "rec-pp-ocrv6-tiny", "/m/rec.onnx", "/m/dict.txt"},
		{map[string]string{"model": "pp-ocrv6-tiny"}, "rec-pp-ocrv6-tiny", "", ""},
		{map[string]string{"model": "tesseract"}, "rec-pp-ocrv6-tiny", "", ""},
	} {
		models.updateConfig(tc.values)
		if _, err := models.getEngine(t.Context()); err != nil {
			t.Fatal(err)
		}
		if ids := getRecogniserIDs(got); !slices.Equal(ids, []string{tc.model}) || got.Recognise.Path != tc.path || got.Recognise.Dict != tc.dict {
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
			files, ok := engine.FindRecognisers(option.Value)
			if !ok {
				t.Errorf("the manifest offers %s, which the engine does not run", option.Value)
				continue
			}
			var provisioned []string
			for _, id := range option.Dependencies {
				provisioned = append(provisioned, sources[id])
			}
			var fetched []string
			for _, file := range files {
				fetched = append(fetched, file.Address)
			}
			if option.Value != engine.RecogniserCustom && !slices.Equal(provisioned, fetched) {
				t.Errorf("%s provisions %v, and the engine fetches %v", option.Value, provisioned, fetched)
			}
		}
	}
	if count == 0 {
		t.Fatal("the manifest offers no model")
	}
}

// getManifestCommandTitle is the title the manifest gives the command id, and false for a command
// the manifest does not declare.
func getManifestCommandTitle(t *testing.T, id string) (string, bool) {
	t.Helper()
	var manifest struct {
		Contributes struct {
			Commands []struct {
				ID           string   `json:"id"`
				Title        string   `json:"title"`
				ContentTypes []string `json:"contentTypes"`
			} `json:"commands"`
		} `json:"contributes"`
	}
	if err := json.Unmarshal(ocr.Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, one := range manifest.Contributes.Commands {
		if one.ID == id {
			if !slices.Equal(one.ContentTypes, []string{"application/pdf"}) {
				t.Errorf("the manifest offers %s for %v, want PDFs only", id, one.ContentTypes)
			}
			return one.Title, true
		}
	}
	return "", false
}

// commandAnswer is the answer of the plugin's command service in the JSON of Connect.
type commandAnswer struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// commandService is the plugin mounted over a vault as the host links it. opens counts the
// recognisers the plugin opened.
type commandService struct {
	url   string
	opens int
}

func newCommandService(t *testing.T, v *hosttest.Vault, texts *hosttest.Texts) *commandService {
	t.Helper()
	plugin := &commandService{url: "", opens: 0}
	models := newEngines(engine.Defaults(), func(context.Context, engine.Config) (recogniser, error) {
		plugin.opens++
		return closing{MockRecogniser: engine.MockRecogniser{}, o: &opened{opened: 0, closed: 0}}, nil
	})
	h, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "tok"}, func(s *server.Server) error {
		return mount(s, recognition.Host{Vault: v, Store: v.Store(), Texts: texts, Queue: hosttest.NewQueue()}, models)
	})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	plugin.url = srv.URL
	return plugin
}

// runCommand runs the command over the file as the host does, and returns the plugin's answer.
func (c *commandService) runCommand(t *testing.T, id, path string) commandAnswer {
	t.Helper()
	body, err := json.Marshal(map[string]string{"command": id, "path": path})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, c.url+"/numen.v1.PluginCommandService/RunCommand", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer tok")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("%s answered %s", id, res.Status)
	}
	var answer commandAnswer
	if err := json.NewDecoder(res.Body).Decode(&answer); err != nil {
		t.Fatal(err)
	}
	return answer
}

// newTextLayerVault is a vault holding books/layer.pdf, a PDF of two pages with a text layer.
func newTextLayerVault(t *testing.T) (*hosttest.Vault, *hosttest.Texts) {
	t.Helper()
	v := hosttest.New("Physics")
	raw, err := os.ReadFile("../document/testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	v.File("books/layer.pdf", string(raw))
	texts := hosttest.NewTexts("Physics", PluginID)
	texts.SetStore(PluginID, v.Store())
	texts.File("books/layer.pdf", raw)
	return v, texts
}

func TestRecogniseOverATextLayerKeepsItAndNamesTheReplacingCommand(t *testing.T) {
	v, texts := newTextLayerVault(t)
	plugin := newCommandService(t, v, texts)
	const title = "Recognise text over the text layer"

	answer := plugin.runCommand(t, "recognition.create", "books/layer.pdf")
	want := `books/layer.pdf has a text layer, which stays the text of its 2 pages. Run "` + title + `" to replace it with a recognition`
	if answer.Error != "" || answer.Message != want || plugin.opens != 0 {
		t.Errorf("Recognise answered %+v, opened the recogniser %d times; want %q", answer, plugin.opens, want)
	}
	if declared, isDeclared := getManifestCommandTitle(t, "recognition.replace"); !isDeclared || declared != title {
		t.Errorf("the manifest declares recognition.replace %v, titled %q; the answer names %q", isDeclared, declared, title)
	}
	listing, err := texts.ListVersions(t.Context(), "books/layer.pdf")
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range listing.Versions {
		if version.Key == recognition.VersionKey {
			t.Errorf("a recognition was kept over the text layer: %+v", version)
		}
	}
}

func TestTheReplacingCommandRecognisesOverATextLayer(t *testing.T) {
	v, texts := newTextLayerVault(t)
	plugin := newCommandService(t, v, texts)

	answer := plugin.runCommand(t, "recognition.replace", "books/layer.pdf")
	if answer.Error != "" || answer.Message != "recognised 2 pages of books/layer.pdf" || plugin.opens != 1 {
		t.Fatalf("the replacing command answered %+v, opened the recogniser %d times", answer, plugin.opens)
	}
	listing, err := texts.ListVersions(t.Context(), "books/layer.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if listing.Live == "" {
		t.Fatal("the file has no live version")
	}
	for _, version := range listing.Versions {
		if version.GetID() == listing.Live && version.Key != recognition.VersionKey {
			t.Errorf("the live version is %+v, want the recognition", version)
		}
	}
}

func TestRecogniseOverAPartialTextLayerReportsTheRecognisedAndTheTextLayerPageCounts(t *testing.T) {
	v := hosttest.New("Physics")
	raw, err := os.ReadFile("../document/testdata/mixed.pdf")
	if err != nil {
		t.Fatal(err)
	}
	v.File("books/mixed.pdf", string(raw))
	texts := hosttest.NewTexts("Physics", PluginID)
	texts.SetStore(PluginID, v.Store())
	texts.File("books/mixed.pdf", raw)
	plugin := newCommandService(t, v, texts)

	answer := plugin.runCommand(t, "recognition.create", "books/mixed.pdf")
	const want = "recognised 2 pages of books/mixed.pdf and kept the text layer of 1 page"
	if answer.Error != "" || answer.Message != want || plugin.opens != 1 {
		t.Errorf("Recognise answered %+v, opened the recogniser %d times; want %q", answer, plugin.opens, want)
	}
	listing, err := texts.ListVersions(t.Context(), "books/mixed.pdf")
	if err != nil {
		t.Fatal(err)
	}
	isLive := false
	for _, version := range listing.Versions {
		isLive = isLive || (version.GetID() == listing.Live && version.Key == recognition.VersionKey)
	}
	if !isLive {
		t.Errorf("the live version of books/mixed.pdf is %q in %+v, want the recognition", listing.Live, listing.Versions)
	}
}

func TestTheMockRecogniserReadsTheModelOfTheSettings(t *testing.T) {
	t.Setenv(EnvMockML, "true")
	cfg := engine.Defaults()
	files, ok := engine.FindRecognisers("pp-ocrv5-cyrillic")
	if !ok {
		t.Fatal("pp-ocrv5-cyrillic is not a model the engine runs")
	}
	cfg.Recognise.Files = files
	opened, err := newEngineOpener(engine.NewLibrary())(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := opened.GetRecognitionModel().Recogniser; got != "rec-pp-ocrv5-cyrillic" {
		t.Errorf("the mock records the recognition as read with %q, want rec-pp-ocrv5-cyrillic", got)
	}
	blocks, err := opened.Recognise(t.Context(), image.NewGray(image.Rect(0, 0, 200, 100)))
	if err != nil || len(blocks) != 1 || !strings.Contains(blocks[0].Text, "rec-pp-ocrv5-cyrillic") {
		t.Errorf("the mock read %+v (%v), want a text that names rec-pp-ocrv5-cyrillic", blocks, err)
	}
}

// getRecogniserIDs is the ids of the recognition models a configuration opens.
func getRecogniserIDs(cfg engine.Config) []string {
	ids := make([]string, 0, len(cfg.Recognise.Files))
	for _, file := range cfg.Recognise.Files {
		ids = append(ids, file.ID)
	}
	return ids
}

func TestAChangedModelClosesTheOpenedRecogniserAndOpensTheNewOne(t *testing.T) {
	o := &opened{}
	var got engine.Config
	models := newEngines(engine.Defaults(), func(ctx context.Context, cfg engine.Config) (recogniser, error) {
		got = cfg
		return o.open(ctx, cfg)
	})
	if _, err := models.getEngine(t.Context()); err != nil {
		t.Fatal(err)
	}
	multilingual := []string{"rec-pp-ocrv5-latin", "rec-pp-ocrv5-cyrillic", "rec-pp-ocrv5-greek", "rec-pp-ocrv6-tiny"}
	if ids := getRecogniserIDs(got); !slices.Equal(ids, multilingual) {
		t.Fatalf("a fresh profile opens %v, want %v", ids, multilingual)
	}

	models.updateConfig(map[string]string{"model": "pp-ocrv5-cyrillic"})
	if o.closed != 1 {
		t.Fatalf("%d recognisers closed after the model changed, want 1", o.closed)
	}
	if _, err := models.getEngine(t.Context()); err != nil {
		t.Fatal(err)
	}
	if ids := getRecogniserIDs(got); o.opened != 2 || !slices.Equal(ids, []string{"rec-pp-ocrv5-cyrillic"}) {
		t.Fatalf("the next page opened %v (%d opened), want rec-pp-ocrv5-cyrillic alone", ids, o.opened)
	}

	for _, value := range []string{"pp-ocrv5-cyrillic", "bogus"} {
		models.updateConfig(map[string]string{"model": value})
		if _, err := models.getEngine(t.Context()); err != nil {
			t.Fatal(err)
		}
		if ids := getRecogniserIDs(got); o.closed != 1 || o.opened != 2 || !slices.Equal(ids, []string{"rec-pp-ocrv5-cyrillic"}) {
			t.Errorf("model %q: %d closed, %d opened, reading with %v; want nothing changed", value, o.closed, o.opened, ids)
		}
	}
}
