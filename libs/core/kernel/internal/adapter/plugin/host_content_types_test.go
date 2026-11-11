package plugin

import (
	"reflect"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
	catalogmanifest "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/contenttype"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func newReaderHost(manifests ...catalogmanifest.Manifest) *Host {
	host := NewHost(testsupport.GetMachine(), "desktop.editor")
	host.ContentTypes.SetContentTypeTable(contenttype.Open())
	for _, manifest := range manifests {
		manifest.Frontend = &catalogmanifest.Frontend{}
		setPlugin(host, manifest.ID, catalogmanifest.Discovered{Manifest: manifest})
	}
	return host
}

func newReader(pluginID, readerID string, rank domain.HandlerRank, types ...domain.ContentType) catalogmanifest.Manifest {
	return catalogmanifest.Manifest{ID: pluginID, Contributes: catalogmanifest.Contributes{
		Readers: []catalogmanifest.Reader{{ID: readerID, ContentTypes: types, Rank: rank}},
	}}
}

func getReaders(host *Host) map[string][]plugindomain.ReaderInfo {
	out := map[string][]plugindomain.ReaderInfo{}
	for _, info := range host.Registry.GetPluginInfos() {
		out[info.ID] = info.Readers
	}
	return out
}

func TestGetReaders_AssignsDefaultTypeToLowerPluginID(t *testing.T) {
	host := newReaderHost(
		newReader("org.example.b", "pdf", domain.RankDefault, "application/pdf"),
		newReader("org.example.a", "pdf", domain.RankDefault, "application/pdf"),
	)

	got := getReaders(host)
	want := map[string][]plugindomain.ReaderInfo{
		"org.example.a": {{ID: "pdf", Extensions: []string{"pdf"}}},
		"org.example.b": {{ID: "pdf"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("readers = %+v; want %+v", got, want)
	}

	host.Settings.SetSettings(map[string]map[string]any{"org.example.a": {settingEnabled: false}})
	if got := getReaders(host)["org.example.b"]; !reflect.DeepEqual(got, []plugindomain.ReaderInfo{{ID: "pdf", Extensions: []string{"pdf"}}}) {
		t.Fatalf("with the lower id disabled, readers = %+v", got)
	}
}

func TestPluginInfosAssignAnExtensionToTheNarrowerType(t *testing.T) {
	host := newReaderHost(
		newReader("org.example.a", "any-audio", domain.RankDefault, "audio/*"),
		newReader("org.example.z", "mp3", domain.RankDefault, "audio/mpeg"),
		newReader("org.example.y", "raw", domain.RankAlternate, "audio/flac"),
	)

	got := getReaders(host)
	audio := got["org.example.a"][0].Extensions
	if !slices.Contains(audio, "wav") || !slices.Contains(audio, "flac") || slices.Contains(audio, "mp3") {
		t.Errorf("audio/* opens %v; want wav and flac and no mp3", audio)
	}
	if mp3 := got["org.example.z"][0].Extensions; !slices.Contains(mp3, "mp3") {
		t.Errorf("audio/mpeg opens %v; want mp3", mp3)
	}
	if raw := got["org.example.y"][0].Extensions; len(raw) != 0 {
		t.Errorf("an alternate reader opens %v; want none", raw)
	}
}
