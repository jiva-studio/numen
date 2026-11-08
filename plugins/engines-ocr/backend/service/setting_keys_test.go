package service

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/engine"
)

func TestCustomModelFilesAreReadFromTheCamelCaseKeys(t *testing.T) {
	models := newEngines(engine.Defaults(), (&opened{}).open)
	models.updateConfig(map[string]string{
		"model": "custom", "customModelPath": "/m/rec.onnx", "customDictPath": "/m/dict.txt",
	})

	got := models.cfg.Recognize
	if got.Path != "/m/rec.onnx" || got.Dict != "/m/dict.txt" {
		t.Errorf("model path %q and dictionary %q, want /m/rec.onnx and /m/dict.txt", got.Path, got.Dict)
	}
}

func TestCustomModelFilesAreNotReadFromTheSnakeCaseKeys(t *testing.T) {
	models := newEngines(engine.Defaults(), (&opened{}).open)
	models.updateConfig(map[string]string{
		"model": "custom", "custom_model_path": "/m/rec.onnx", "custom_dict_path": "/m/dict.txt",
	})

	got := models.cfg.Recognize
	if got.Path != "" || got.Dict != "" {
		t.Errorf("model path %q and dictionary %q, want none", got.Path, got.Dict)
	}
}
