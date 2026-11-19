package plugin_test

import (
	"slices"
	"testing"

	catalogmanifest "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/protocol"
)

func takerOf(contentType domain.ContentType) catalogmanifest.Manifest {
	return catalogmanifest.Manifest{ActivationEvents: []string{catalogmanifest.ActivationEventFilesChanged + ":" + string(contentType)}}
}

func TestFilesChangedWakesOnlyThePluginsThatTakeTheChangedTypes(t *testing.T) {
	pdf, epub, tags := &eventRecorder{}, &eventRecorder{}, &eventRecorder{}
	host := newEventHost(t,
		map[string]catalogmanifest.Manifest{
			"md.test.pdf":  takerOf("application/pdf"),
			"md.test.epub": takerOf("application/epub+zip"),
			"md.test.tags": {ActivationEvents: []string{catalogmanifest.ActivationEventFilesChanged}},
		},
		map[string]int{
			"md.test.pdf":  servePlugin(t, pdf),
			"md.test.epub": servePlugin(t, epub),
			"md.test.tags": servePlugin(t, tags),
		},
	)

	host.Management.Events.EmitFilesChanged("vault-1", []domain.FileType{{Content: host.Management.ContentTypes.GetContentType("books/river.pdf")}})
	if err := host.Management.Events.GetEventBus().Flush(t.Context()); err != nil {
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
		t.Errorf("the plugin with no mask received %v, want [%s]", got, event)
	}
}

func TestIsTakingFileMatchesTheMasksOfTheActivationEvent(t *testing.T) {
	host := newEventHost(t,
		map[string]catalogmanifest.Manifest{
			"md.test.pdf":  takerOf("application/pdf"),
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
		if got := host.Management.ContentTypes.IsTakingFile(tc.pluginID, tc.path, nil); got != tc.want {
			t.Errorf("%s takes %s: %v, want %v", tc.pluginID, tc.path, got, tc.want)
		}
	}
}

func TestNoteTypeMaskSelectsEventsAndFeedRows(t *testing.T) {
	const id = "md.test.articles"
	recorder := &eventRecorder{}
	host := newEventHost(t,
		map[string]catalogmanifest.Manifest{id: {ActivationEvents: []string{"onFilesChanged:note:article"}}},
		map[string]int{id: servePlugin(t, recorder)},
	)
	for _, kind := range []domain.NoteType{domain.TypeNote, "article"} {
		host.Management.Events.EmitFilesChanged("vault-1", []domain.FileType{{Content: "text/markdown", Note: kind}})
		if err := host.Management.Events.GetEventBus().Flush(t.Context()); err != nil {
			t.Fatal(err)
		}
		if got := host.Management.ContentTypes.IsTakingFile(id, "paper.md", []domain.NoteType{kind}); got != (kind == "article") {
			t.Errorf("matches feed row of type %q = %v", kind, got)
		}
	}
	event := protocol.GetFilesChangedEvent("vault-1")
	if _, err := host.Management.Events.TriggerEvent(t.Context(), event); err != nil {
		t.Fatal(err)
	}
	if got := recorder.getEvents(); !slices.Equal(got, []string{event}) {
		t.Errorf("events = %v, want a single article change", got)
	}
}
