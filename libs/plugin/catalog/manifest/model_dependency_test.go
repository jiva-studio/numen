package manifest_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

var (
	encoderSum  = strings.Repeat("e", 64)
	decoderSum  = strings.Repeat("d", 64)
	detectorSum = strings.Repeat("c", 64)
)

func readModelManifest(t *testing.T) manifest.Manifest {
	t.Helper()
	m, err := manifestjson.Read(strings.NewReader(`{
		"id": "md.example.speech", "name": "Speech", "version": "1.0.0",
		"backend": {"entrypoint": "bin/speech"},
		"dependencies": [
			{"id": "speech-runtime", "kind": "runtime", "environment": ["SPEECH_LIB"]},
			{"id": "speech-tool", "kind": "binary"},
			{"id": "large-encoder", "kind": "model", "version": "r1",
				"sourceUrls": ["https://models.example.org/large/encoder.onnx", "https://mirror.example.org/large/encoder.onnx"],
				"checksumSha256": "` + encoderSum + `", "sizeBytes": 4096, "metadata": {"title": "Large encoder"}},
			{"id": "small-decoder", "kind": "model", "version": "r1",
				"sourceUrl": "https://models.example.org/small/decoder.onnx", "checksumSha256": "` + decoderSum + `", "sizeBytes": 2048},
			{"id": "voice-detector", "kind": "model", "version": "r2",
				"sourceUrl": "https://models.example.org/detector.onnx", "checksumSha256": "` + detectorSum + `", "sizeBytes": 512}
		],
		"settings": [
			{"key": "model", "type": "enum", "title": "Model", "default": "large", "options": [
				{"value": "large", "title": "Large", "dependencies": ["large-encoder"]},
				{"value": "small", "title": "Small", "dependencies": ["small-decoder", "speech-tool"]}
			]}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestTheRequirementsOfAPluginHoldNoModelUnderAnyOption(t *testing.T) {
	m := readModelManifest(t)
	for model, want := range map[string][]string{
		"large": {"speech-runtime"},
		"small": {"speech-runtime", "speech-tool"},
	} {
		var got []string
		for _, req := range m.GetDependencyRequirements("/p", m.ResolveSettings(map[string]any{"model": model})) {
			got = append(got, req.ID)
		}
		if !slices.Equal(got, want) {
			t.Errorf("model %q: requirements = %v, want %v", model, got, want)
		}
	}
}

func TestTheSizeOfADependencyIsReadFromTheManifest(t *testing.T) {
	m := readModelManifest(t)
	req, ok := m.FindModelDependency("large-encoder")
	if !ok {
		t.Fatal("large-encoder is not found")
	}
	if req.SizeBytes != 4096 {
		t.Errorf("size = %d, want 4096", req.SizeBytes)
	}
}

func TestAPluginFindsOnlyTheModelsItsManifestDeclares(t *testing.T) {
	m := readModelManifest(t)
	req, ok := m.FindModelDependency("large-encoder")
	if !ok {
		t.Fatal("large-encoder is not found")
	}
	if req.Kind != manifest.DependencyKindModel || req.ChecksumSHA256 != encoderSum ||
		!slices.Equal(req.SourceURLs, []string{"https://models.example.org/large/encoder.onnx", "https://mirror.example.org/large/encoder.onnx"}) {
		t.Errorf("large-encoder = %+v", req)
	}
	for _, id := range []string{"speech-runtime", "speech-tool", "absent-model", ""} {
		if found, ok := m.FindModelDependency(id); ok {
			t.Errorf("%q is found as a model: %+v", id, found)
		}
	}
}

func TestADeclaredModelFileIsStoredByItsChecksumUnderItsTitle(t *testing.T) {
	m := readModelManifest(t)
	for id, title := range map[string]string{"large-encoder": "Large encoder", "small-decoder": "small-decoder"} {
		declared, ok := m.FindModelDependency(id)
		if !ok {
			t.Fatalf("%s is not found", id)
		}
		req, err := manifest.NewDeclaredModelFile(declared)
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if req.ID != declared.ChecksumSHA256 || req.Kind != manifest.DependencyKindModel || req.ChecksumSHA256 != declared.ChecksumSHA256 {
			t.Errorf("%s: requirement = %+v, want the id and checksum %s", id, req, declared.ChecksumSHA256)
		}
		if !slices.Equal(req.SourceURLs, declared.SourceURLs) || req.SourceURL != declared.SourceURL {
			t.Errorf("%s: sources = %q %v, want %q %v", id, req.SourceURL, req.SourceURLs, declared.SourceURL, declared.SourceURLs)
		}
		if got := req.GetTitle(); got != title {
			t.Errorf("%s: title = %q, want %q", id, got, title)
		}
		if req.SizeBytes != declared.SizeBytes {
			t.Errorf("%s: size = %d, want %d", id, req.SizeBytes, declared.SizeBytes)
		}
	}
}

func TestADeclaredModelFileWithoutAChecksumIsRefused(t *testing.T) {
	_, err := manifest.NewDeclaredModelFile(manifest.DependencyRequirement{
		ID: "unpinned", Kind: manifest.DependencyKindModel, SourceURL: "https://models.example.org/unpinned.onnx", SizeBytes: 10,
	})
	if !errors.Is(err, manifest.ErrInvalidDependency) {
		t.Errorf("err = %v, want ErrInvalidDependency", err)
	}
}

func TestTwoPluginsDeclaringOneFileShareOneDownload(t *testing.T) {
	address := "https://models.example.org/shared/graph.onnx"
	first, err := manifest.NewDeclaredModelFile(manifest.DependencyRequirement{
		ID: "shared-graph", Kind: manifest.DependencyKindModel, Version: "r1", SourceURL: address, ChecksumSHA256: encoderSum,
		SizeBytes: 4096, PluginDir: "/plugins/speech", Metadata: map[string]string{"title": "Speech graph"},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := manifest.NewDeclaredModelFile(manifest.DependencyRequirement{
		ID: "graph", Kind: manifest.DependencyKindModel, Version: "r1", SourceURL: address, ChecksumSHA256: encoderSum,
		SizeBytes: 4096, PluginDir: "/plugins/search",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.GetIdentity() != second.GetIdentity() {
		t.Errorf("two declarations of one file are two downloads:\n%q\n%q", first.GetIdentity(), second.GetIdentity())
	}
	other, err := manifest.NewDeclaredModelFile(manifest.DependencyRequirement{
		ID: "graph", Kind: manifest.DependencyKindModel, Version: "r1", SourceURL: address, ChecksumSHA256: decoderSum, SizeBytes: 4096,
	})
	if err != nil {
		t.Fatal(err)
	}
	if other.GetIdentity() == first.GetIdentity() {
		t.Error("two files of different checksums are one download")
	}
}
