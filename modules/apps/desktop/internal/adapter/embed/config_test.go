package embed_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed"
)

// One model name is a repository and something a service answers to, and the
// two are not one set of vectors.
func TestAModelRunHereAndOneServedAreTwoAddresses(t *testing.T) {
	name := "intfloat/multilingual-e5-small"
	here := embed.Station{Use: embed.UseLocal, Local: embed.LocalModel{Name: name}}
	served := embed.Station{
		Use:     embed.UseService,
		Service: embed.ServiceModel{BaseURL: "http://127.0.0.1:1/v1", Name: name},
	}
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

// A station naming neither is an installation with no model, and has no
// address at all.
func TestAStationThatNamesNeitherIsNowhere(t *testing.T) {
	if got := (embed.Station{}).From(); got != "" {
		t.Errorf("got %q", got)
	}
}

// What a vector is kept under is the model, made where the index is filled. A
// question placed elsewhere claims those rows.
func TestVectorsAreKeptUnderTheStationThatFillsTheIndex(t *testing.T) {
	cfg := embed.Defaults()
	cfg.Query.Use = embed.UseService

	if got := cfg.Stored().From; got != cfg.Indexing.From() {
		t.Errorf("kept under %q, filled at %q", got, cfg.Indexing.From())
	}
	if got := cfg.Stored().Name; got != cfg.Model.Name {
		t.Errorf("kept under %q, and the model is %q", got, cfg.Model.Name)
	}
}
