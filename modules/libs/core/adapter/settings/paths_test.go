package settings_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
)

// Every path the settings page reads is a path through the file this build
// writes. A row reading a key nothing writes draws nothing, whatever the file
// holds.
func TestEveryPathThePageReadsStandsInTheFile(t *testing.T) {
	written, err := settings.Written(settings.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	var held any
	if err := json.Unmarshal([]byte(written), &held); err != nil {
		t.Fatal(err)
	}

	for _, at := range []string{
		"indexing.embedding.model.name",
		"indexing.recognition.recognise.name",
		"indexing.recognition.proofread.with",
		"indexing.recognition.proofread.automatically",
		"indexing.transcribe_recordings",
		"indexing.transcribe_under_mb",
		"indexing.transcription.proofread.with",
		"indexing.transcription.proofread.automatically",
		"indexing.proofreading.profiles",
		"agent.use",
		"agent.claude.model",
		"agent.claude.max_steps",
		"agent.serve_tools",
		"agent.claude.reads_hooks_and_skills",
	} {
		value := held
		ok := true
		for _, step := range strings.Split(at, ".") {
			object, is := value.(map[string]any)
			if !is {
				ok = false
				break
			}
			value, ok = object[step]
			if !ok {
				break
			}
		}
		if !ok {
			t.Errorf("%s is no path through the file", at)
		}
	}
}
