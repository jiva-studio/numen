package plugin_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/protocol"
	"github.com/jiva-studio/numen/libs/note/domain"
	catalogmanifest "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func readerOf(contentType domain.ContentType) catalogmanifest.Manifest {
	return catalogmanifest.Manifest{
		ActivationEvents: []string{catalogmanifest.ActivationEventFilesChanged},
		Contributes: catalogmanifest.Contributes{
			Readers: []catalogmanifest.Reader{{ID: "reader", ContentTypes: []domain.ContentType{contentType}, Rank: domain.RankDefault}},
		},
	}
}

func TestFilesChangedWakesOnlyThePluginsThatTakeTheChangedTypes(t *testing.T) {
	pdf, epub, tags := &eventRecorder{}, &eventRecorder{}, &eventRecorder{}
	host := newEventHost(t,
		map[string]catalogmanifest.Manifest{
			"md.test.pdf":  readerOf("application/pdf"),
			"md.test.epub": readerOf("application/epub+zip"),
			"md.test.tags": {ActivationEvents: []string{catalogmanifest.ActivationEventFilesChanged}},
		},
		map[string]int{
			"md.test.pdf":  servePlugin(t, pdf),
			"md.test.epub": servePlugin(t, epub),
			"md.test.tags": servePlugin(t, tags),
		},
	)

	host.Events.EmitFilesChanged("vault-1", []domain.ContentType{host.ContentTypes.GetContentType("books/river.pdf")})
	if err := host.Events.GetEventBus().Flush(t.Context()); err != nil {
		t.Fatal(err)
	}

	event := protocol.GetFilesChangedEvent("vault-1")
	if got := pdf.getEvents(); !slices.Equal(got, []string{event}) {
		t.Errorf("the plugin that takes pdf received %v, want [%s]", got, event)
	}
	if got := epub.getEvents(); len(got) != 0 {
		t.Errorf("the plugin that takes epub received %v for a pdf", got)
	}
	if got := tags.getEvents(); !slices.Equal(got, []string{event}) {
		t.Errorf("the plugin with no reader received %v, want [%s]", got, event)
	}
}

func TestAPluginTakesTheFilesOfTheTypesItsReadersOpen(t *testing.T) {
	host := newEventHost(t,
		map[string]catalogmanifest.Manifest{
			"md.test.pdf":  readerOf("application/pdf"),
			"md.test.tags": {ActivationEvents: []string{catalogmanifest.ActivationEventFilesChanged}},
		},
		map[string]int{},
	)
	for _, tc := range []struct {
		pluginID, path string
		want           bool
	}{
		{"md.test.pdf", "books/river.pdf", true},
		{"md.test.pdf", "books/lake.epub", false},
		{"md.test.tags", "books/lake.epub", true},
	} {
		if got := host.ContentTypes.IsTakingFile(tc.pluginID, tc.path); got != tc.want {
			t.Errorf("%s takes %s: %v, want %v", tc.pluginID, tc.path, got, tc.want)
		}
	}
}
