package pluginport_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

const readerManifestHead = `{"id":"org.example.reader","name":"A","version":"1.0.0","frontend":{"entrypoint":"dist/index.js"}`

func TestReadManifestReadsReadersAndNewContentTypes(t *testing.T) {
	m, err := pluginport.ReadManifest(strings.NewReader(readerManifestHead + `,
		"contentTypes": [{"type": "application/x-example-sample", "globs": ["*.sample"]}],
		"readers": [
			{"id": "sample", "contentTypes": ["application/x-example-sample"], "rank": "default"},
			{"id": "audio", "contentTypes": ["audio/*"], "rank": "alternate"}
		]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.ContentHandler{
		{PluginID: "org.example.reader", ID: "sample", Type: "application/x-example-sample", Rank: domain.RankDefault},
		{PluginID: "org.example.reader", ID: "audio", Type: "audio/*", Rank: domain.RankAlternate},
	}
	if got := m.ListContentHandlers(); !slices.Equal(got, want) {
		t.Errorf("ListContentHandlers = %+v; want %+v", got, want)
	}
	if got := m.ContentTypes[0].ListExtensions(); !slices.Equal(got, []string{"sample"}) {
		t.Errorf("ListExtensions = %v; want [sample]", got)
	}
	if !m.IsHandling("audio/mpeg") || m.IsHandling("video/mp4") {
		t.Error("IsHandling answers for the wrong types")
	}
}

func TestIsHandlingMatchesTheTypesOfTheTextsItMakes(t *testing.T) {
	m, err := pluginport.ReadManifest(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","backend":{"entrypoint":"bin/a"},
		"texts": {"contentTypes": ["audio/*", "video/*"]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !m.IsHandling("video/mp4") || m.IsHandling("application/pdf") {
		t.Error("IsHandling answers for the wrong types")
	}
}

func TestReadManifestRejectsAnInvalidReaderOrContentType(t *testing.T) {
	cases := map[string]struct {
		fields string
		want   error
	}{
		"a reader with no rank":    {`"readers":[{"id":"r","contentTypes":["application/pdf"]}]`, pluginport.ErrInvalidReader},
		"a reader of no type":      {`"readers":[{"id":"r","contentTypes":[],"rank":"default"}]`, pluginport.ErrInvalidReader},
		"a reader with no id":      {`"readers":[{"contentTypes":["application/pdf"],"rank":"default"}]`, pluginport.ErrInvalidReader},
		"a reader declared twice":  {`"readers":[{"id":"r","contentTypes":["text/plain"],"rank":"default"},{"id":"r","contentTypes":["text/plain"],"rank":"default"}]`, pluginport.ErrInvalidReader},
		"a reader of an extension": {`"readers":[{"id":"r","contentTypes":[".pdf"],"rank":"default"}]`, pluginport.ErrInvalidReader},
		"a new type as a mask":     {`"contentTypes":[{"type":"audio/*","globs":["*.x"]}]`, domain.ErrInvalidContentType},
		"a new type of no glob":    {`"contentTypes":[{"type":"application/x-a","globs":[]}]`, domain.ErrInvalidContentType},
		"a glob of a whole name":   {`"contentTypes":[{"type":"application/x-a","globs":["sample"]}]`, domain.ErrInvalidContentType},
		"a glob in capitals":       {`"contentTypes":[{"type":"application/x-a","globs":["*.SAMPLE"]}]`, domain.ErrInvalidContentType},
	}
	for name, one := range cases {
		_, err := pluginport.ReadManifest(strings.NewReader(readerManifestHead + "," + one.fields + "}"))
		if !errors.Is(err, one.want) {
			t.Errorf("%s: ReadManifest = %v; want %v", name, err, one.want)
		}
	}
}

func TestReadManifestRejectsAReaderWithoutFrontend(t *testing.T) {
	_, err := pluginport.ReadManifest(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0",
		"readers":[{"id":"r","contentTypes":["application/pdf"],"rank":"default"}]}`))
	if !errors.Is(err, pluginport.ErrInvalidReader) {
		t.Fatalf("ReadManifest = %v; want ErrInvalidReader", err)
	}
}

func TestReadManifestReadsSettingsViews(t *testing.T) {
	m, err := pluginport.ReadManifest(strings.NewReader(readerManifestHead + `,
		"settings": [{"key": "model", "type": "string", "title": "Model"}],
		"settingsViews": [
			{"id": "floor", "title": "Search relevance", "after": "model"},
			{"id": "top", "title": "Top"}
		]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := []pluginport.PluginSettingsView{
		{ID: "floor", Title: "Search relevance", After: "model"},
		{ID: "top", Title: "Top"},
	}
	if !slices.Equal(m.SettingsViews, want) {
		t.Errorf("SettingsViews = %+v; want %+v", m.SettingsViews, want)
	}
}

func TestReadManifestRejectsAnInvalidSettingsView(t *testing.T) {
	cases := map[string]string{
		"a view with no id":         `"settingsViews":[{"title":"T"}]`,
		"a view declared twice":     `"settingsViews":[{"id":"v","title":"T"},{"id":"v","title":"U"}]`,
		"a view with no title":      `"settingsViews":[{"id":"v"}]`,
		"a view after an unknown":   `"settingsViews":[{"id":"v","title":"T","after":"nope"}]`,
		"a view after a secret":     `"settingsViews":[{"id":"v","title":"T","after":"apiKey"}]`,
		"a view after its own id":   `"settingsViews":[{"id":"v","title":"T","after":"v"}]`,
		"a view with a blank title": `"settingsViews":[{"id":"v","title":"  "}]`,
	}
	for name, fields := range cases {
		_, err := pluginport.ReadManifest(strings.NewReader(readerManifestHead + "," + fields + "}"))
		if !errors.Is(err, pluginport.ErrInvalidSettingsView) {
			t.Errorf("%s: ReadManifest = %v; want ErrInvalidSettingsView", name, err)
		}
	}
}

func TestReadManifestRejectsASettingsViewWithoutFrontend(t *testing.T) {
	_, err := pluginport.ReadManifest(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0",
		"settingsViews":[{"id":"v","title":"T"}]}`))
	if !errors.Is(err, pluginport.ErrInvalidSettingsView) {
		t.Fatalf("ReadManifest = %v; want ErrInvalidSettingsView", err)
	}
}
