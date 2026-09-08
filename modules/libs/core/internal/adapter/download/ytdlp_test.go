package download

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// One language is asked for and not a list. A site that publishes a machine
// translation into every language it knows answers a request for the lot by
// refusing it.
func TestWhichLanguageTheWordsAreAskedFor(t *testing.T) {
	for _, one := range []struct {
		found     port.Metadata
		languages []string
		automatic bool
		want      string
	}{
		{port.Metadata{Captions: []string{"de", "en", "ru"}}, []string{"ru", "en"}, true, "ru"},
		{port.Metadata{Captions: []string{"de", "en"}}, []string{"ru"}, true, "de"},
		{port.Metadata{Automatic: []string{"ab", "en-orig", "zu"}}, nil, true, "en-orig"},
		{port.Metadata{Automatic: []string{"ab", "en-orig"}}, []string{"en"}, true, "en-orig"},
		{port.Metadata{Automatic: []string{"ab", "en-orig"}}, nil, false, ""},
		{port.Metadata{Captions: []string{"en"}, Automatic: []string{"ru-orig"}}, nil, true, "en"},
		{port.Metadata{}, []string{"en"}, true, ""},
	} {
		if got := language(one.found, one.languages, one.automatic); got != one.want {
			t.Errorf("%+v with %v: asked for %q, want %q",
				one.found, one.languages, got, one.want)
		}
	}
}

// A video somebody translated into thirty languages publishes words in all
// thirty, and what was said in it is one of them.
func TestAVideoTranslatedIntoManyLanguages(t *testing.T) {
	meta := port.Metadata{
		Language: "en",
		Captions: []string{"ar", "de", "en", "es", "zh"},
	}
	if got := language(meta, nil, true); got != "en" {
		t.Errorf("asked for %q, want the language it was spoken in", got)
	}
	if got := language(meta, []string{"de"}, true); got != "de" {
		t.Errorf("asked for %q, want the language this installation named", got)
	}
}
