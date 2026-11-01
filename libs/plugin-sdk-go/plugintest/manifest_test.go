package plugintest

import "testing"

func TestReadManifestReadsWhatTheHostReads(t *testing.T) {
	raw := []byte(`{
		"id": "org.example.sample",
		"name": "Sample",
		"version": "1.0.0",
		"protocol": 1,
		"targets": ["desktop.editor"],
		"backend": {"entrypoint": "bin/sample"},
		"settings": [{"key": "model", "type": "enum", "title": "Model", "default": "small",
			"options": [{"value": "small"}, {"value": "large"}]}]
	}`)

	m := ReadManifest(t, raw)

	if m.ID != "org.example.sample" || m.Backend.Entrypoint != "bin/sample" {
		t.Errorf("got id %q and entrypoint %q", m.ID, m.Backend.Entrypoint)
	}
	if len(m.Settings) != 1 || len(m.Settings[0].Options) != 2 || m.Settings[0].Options[1].Value != "large" {
		t.Errorf("got settings %+v", m.Settings)
	}
}
