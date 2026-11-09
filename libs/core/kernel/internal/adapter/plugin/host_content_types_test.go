package plugin

import (
	"reflect"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/sharedmimeinfo"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

func newReaderHost(manifests ...pluginport.PluginManifest) *Host {
	host := NewHost(testsupport.GetMachine(), "desktop.editor")
	host.SetContentTypeTable(sharedmimeinfo.Open())
	for _, manifest := range manifests {
		manifest.Frontend = &pluginport.PluginFrontend{}
		host.plugins[manifest.ID] = pluginport.DiscoveredPlugin{Manifest: manifest}
	}
	return host
}

func newReader(pluginID, readerID string, rank domain.HandlerRank, types ...domain.ContentType) pluginport.PluginManifest {
	return pluginport.PluginManifest{ID: pluginID, Contributes: pluginport.PluginContributes{
		Readers: []pluginport.PluginReader{{ID: readerID, ContentTypes: types, Rank: rank}},
	}}
}

func getReaders(host *Host) map[string][]pluginport.PluginReaderInfo {
	out := map[string][]pluginport.PluginReaderInfo{}
	for _, info := range host.GetPluginInfos() {
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
	want := map[string][]pluginport.PluginReaderInfo{
		"org.example.a": {{ID: "pdf", Extensions: []string{"pdf"}}},
		"org.example.b": {{ID: "pdf"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("readers = %+v; want %+v", got, want)
	}

	host.SetSettings(map[string]map[string]any{"org.example.a": {settingEnabled: false}})
	if got := getReaders(host)["org.example.b"]; !reflect.DeepEqual(got, []pluginport.PluginReaderInfo{{ID: "pdf", Extensions: []string{"pdf"}}}) {
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
