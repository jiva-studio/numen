package embed_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
)

// One model name is a repository and something a service answers to, and the
// two are not one set of vectors.
func TestAModelRunHereAndOneServedAreTwoAddresses(t *testing.T) {
	name := "intfloat/multilingual-e5-small"
	here := embed.Provider{}.Running(embed.LocalModel{Name: name})
	served := embed.Provider{}.Serving(embed.ServiceModel{BaseURL: "http://127.0.0.1:1/v1", Name: name})
	if a, b := here.From(), served.From(); a == b {
		t.Errorf("both are %q", a)
	}
}

// Two services answering to one name are two addresses, and one service asked
// for two models is two more.
func TestAServiceIsAddressedByWhereItIsAndWhatIsAskedOfIt(t *testing.T) {
	at := func(baseURL, name string) string {
		return embed.ServiceModel{BaseURL: baseURL, Name: name}.From()
	}
	if a, b := at("https://api.openai.com/v1", "e5"), at("http://127.0.0.1:1/v1", "e5"); a == b {
		t.Errorf("two services are both %q", a)
	}
	if a, b := at("https://api.openai.com/v1", "e5"), at("https://api.openai.com/v1", "bge"); a == b {
		t.Errorf("two models are both %q", a)
	}
	// One address written two ways is one address.
	if a, b := at("https://api.openai.com/v1/", "e5"), at("https://api.openai.com/v1", "e5"); a != b {
		t.Errorf("%q and %q", a, b)
	}
}

// A model on this machine is addressed by the weights that are run, so the
// configurations that name one file are one address.
func TestOneBuildOnThisMachineIsOneAddress(t *testing.T) {
	repository := "intfloat/multilingual-e5-small"
	named := embed.LocalModel{Name: repository, File: embed.ModelFile}.From()
	if got := (embed.LocalModel{Name: repository}).From(); got != named {
		t.Errorf("%q and %q", got, named)
	}
	// Another build of one repository is another set of vectors.
	if got := (embed.LocalModel{Name: repository, File: "model_int8.onnx"}).From(); got == named {
		t.Errorf("two builds are both %q", got)
	}
	// A folder is what the weights are read from, and is written how it is
	// written.
	held := embed.LocalModel{Dir: "/models/e5"}.From()
	if got := (embed.LocalModel{Dir: "/models/e5/"}).From(); got != held {
		t.Errorf("%q and %q", got, held)
	}
	if held == named {
		t.Errorf("a folder and a repository are both %q", held)
	}
}

// A provider naming neither is an installation with no model, and has no
// address at all.
func TestAProviderThatNamesNeitherIsNowhere(t *testing.T) {
	if got := (embed.Provider{}).From(); got != "" {
		t.Errorf("got %q", got)
	}
}

// The settings behind a provider are not fields anybody can name, so the keys a
// file already on somebody's disk is read and written by are written down.
func TestAProviderIsReadAndWrittenByTheKeysAFileAlreadyHas(t *testing.T) {
	var held embed.Provider
	if err := json.Unmarshal(
		[]byte(`{"use":"service","local":{"name":"a/repository"},"service":{"name":"served"}}`),
		&held,
	); err != nil {
		t.Fatal(err)
	}
	service, ok := held.Service()
	if !ok || service.Name != "served" {
		t.Fatalf("read as %+v", held)
	}

	written, err := json.Marshal(held)
	if err != nil {
		t.Fatal(err)
	}
	var back map[string]json.RawMessage
	if err := json.Unmarshal(written, &back); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"use", "local", "service"} {
		if _, named := back[key]; !named {
			t.Errorf("%q is not a key of %s", key, written)
		}
	}
	if len(back) != 3 {
		t.Errorf("a provider is written as %s", written)
	}
	// The half the file is not on is kept, so a person who goes back to it
	// finds what they left.
	if !strings.Contains(string(written), "a/repository") {
		t.Errorf("the repository is gone from %s", written)
	}
}
