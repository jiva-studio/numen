package manifestjson_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func readDependency(dependency string) error {
	_, err := manifestjson.Read(strings.NewReader(
		`{"id":"md.example.speech","name":"Speech","version":"1.0.0","backend":{"entrypoint":"bin/speech"},"dependencies":[` + dependency + `]}`,
	))
	return err
}

func TestAManifestRejectsAModelWithoutItsSize(t *testing.T) {
	sum := strings.Repeat("a", 64)
	for name, dependency := range map[string]string{
		"no size":       `{"id":"encoder","kind":"model","sourceUrl":"https://models.example.org/encoder.onnx","checksumSha256":"` + sum + `"}`,
		"zero size":     `{"id":"encoder","kind":"model","sourceUrl":"https://models.example.org/encoder.onnx","checksumSha256":"` + sum + `","sizeBytes":0}`,
		"negative size": `{"id":"encoder","kind":"model","sourceUrl":"https://models.example.org/encoder.onnx","checksumSha256":"` + sum + `","sizeBytes":-1}`,
		"no checksum":   `{"id":"encoder","kind":"model","sourceUrl":"https://models.example.org/encoder.onnx","sizeBytes":4096}`,
		"mirrors, no size": `{"id":"encoder","kind":"model","sourceUrls":["https://models.example.org/encoder.onnx","https://mirror.example.org/encoder.onnx"],` +
			`"checksumSha256":"` + sum + `"}`,
	} {
		if err := readDependency(dependency); !errors.Is(err, manifest.ErrInvalidDependency) {
			t.Errorf("%s: err = %v, want ErrInvalidDependency", name, err)
		}
	}
}

func TestAManifestAcceptsASizedModelAndADependencyOfAnotherKindWithoutASize(t *testing.T) {
	sum := strings.Repeat("a", 64)
	for name, dependency := range map[string]string{
		"sized model":        `{"id":"encoder","kind":"model","sourceUrl":"https://models.example.org/encoder.onnx","checksumSha256":"` + sum + `","sizeBytes":4096}`,
		"model of no source": `{"id":"encoder","kind":"model"}`,
		"runtime":            `{"id":"speech-runtime","kind":"runtime","sourceUrl":"https://example.org/lib.so","checksumSha256":"` + sum + `"}`,
		"sized runtime":      `{"id":"speech-runtime","kind":"runtime","sourceUrl":"https://example.org/lib.so","checksumSha256":"` + sum + `","sizeBytes":100}`,
		"binary":             `{"id":"speech-tool","kind":"binary"}`,
	} {
		if err := readDependency(dependency); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}
