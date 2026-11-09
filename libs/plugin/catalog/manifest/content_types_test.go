package manifest_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

const readerManifestHead = `{"id":"org.example.reader","name":"A","version":"1.0.0","frontend":{}`

func TestReadManifestReadsReaders(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(readerManifestHead + `,
		"contributes": {
			"readers": [
				{"id": "sample", "contentTypes": ["application/x-example-sample"], "rank": "default"},
				{"id": "audio", "contentTypes": ["audio/*"], "rank": "alternate"}
			]}}`))
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
	if !m.IsHandling("audio/mpeg") || m.IsHandling("video/mp4") {
		t.Error("IsHandling answers for the wrong types")
	}
}

func TestReadManifestRejectsAnInvalidReader(t *testing.T) {
	cases := map[string]struct {
		fields string
		want   error
	}{
		"a reader with no rank":    {`"readers":[{"id":"r","contentTypes":["application/pdf"]}]`, manifest.ErrInvalidReader},
		"a reader of no type":      {`"readers":[{"id":"r","contentTypes":[],"rank":"default"}]`, manifest.ErrInvalidReader},
		"a reader with no id":      {`"readers":[{"contentTypes":["application/pdf"],"rank":"default"}]`, manifest.ErrInvalidReader},
		"a reader declared twice":  {`"readers":[{"id":"r","contentTypes":["text/plain"],"rank":"default"},{"id":"r","contentTypes":["text/plain"],"rank":"default"}]`, manifest.ErrInvalidReader},
		"a reader of an extension": {`"readers":[{"id":"r","contentTypes":[".pdf"],"rank":"default"}]`, manifest.ErrInvalidReader},
		"a reader in capitals":     {`"readers":[{"id":"r","contentTypes":["Application/PDF"],"rank":"default"}]`, manifest.ErrInvalidReader},
	}
	for name, one := range cases {
		_, err := manifestjson.Read(strings.NewReader(readerManifestHead + `,"contributes":{` + one.fields + "}}"))
		if !errors.Is(err, one.want) {
			t.Errorf("%s: ReadManifest = %v; want %v", name, err, one.want)
		}
		if name == "a reader of an extension" && (!errors.Is(err, domain.ErrInvalidContentType) || strings.Contains(err.Error(), "not written in lowercase")) {
			t.Errorf("%s: unexpected content type error %v", name, err)
		}
		if name == "a reader in capitals" && (!errors.Is(err, domain.ErrInvalidContentType) || !strings.Contains(err.Error(), "not written in lowercase")) {
			t.Errorf("%s: unexpected uppercase content type error %v", name, err)
		}
	}
}

func TestReadManifestRejectsAReaderWithoutFrontend(t *testing.T) {
	_, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0",
		"contributes":{"readers":[{"id":"r","contentTypes":["application/pdf"],"rank":"default"}]}}`))
	if !errors.Is(err, manifest.ErrInvalidReader) {
		t.Fatalf("ReadManifest = %v; want ErrInvalidReader", err)
	}
}

func TestReadManifestReadsSettingsViews(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(readerManifestHead + `,
		"settings": [{"key": "model", "type": "string", "title": "Model"}],
		"settingsViews": [
			{"id": "floor", "title": "Search relevance"},
			{"id": "top", "title": "Top"}
		]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := []manifest.SettingsView{
		{ID: "floor", Title: "Search relevance"},
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
		"a view with a blank title": `"settingsViews":[{"id":"v","title":"  "}]`,
	}
	for name, fields := range cases {
		_, err := manifestjson.Read(strings.NewReader(readerManifestHead + "," + fields + "}"))
		if !errors.Is(err, manifest.ErrInvalidSettingsView) {
			t.Errorf("%s: ReadManifest = %v; want ErrInvalidSettingsView", name, err)
		}
	}
}

func TestReadManifestRejectsASettingsViewWithoutFrontend(t *testing.T) {
	_, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0",
		"settingsViews":[{"id":"v","title":"T"}]}`))
	if !errors.Is(err, manifest.ErrInvalidSettingsView) {
		t.Fatalf("ReadManifest = %v; want ErrInvalidSettingsView", err)
	}
}
