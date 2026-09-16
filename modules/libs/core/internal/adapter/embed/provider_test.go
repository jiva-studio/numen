package embed_test

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
)

// One model name is a repository and something a service answers to, and the
// two are not one set of vectors.
func TestAModelRunHereAndOneServedAreTwoAddresses(t *testing.T) {
	name := "intfloat/multilingual-e5-small"
	here := embed.Provider{}.SetLocal(embed.LocalModel{Name: name})
	served := embed.Provider{}.SetService(embed.ServiceModel{BaseURL: "http://127.0.0.1:1/v1", Name: name})
	if a, b := here.GetAddress(), served.GetAddress(); a == b {
		t.Errorf("both are %q", a)
	}
}

// Two services answering to one name are two addresses, and one service asked
// for two models is two more.
func TestAServiceIsAddressedByWhereItIsAndWhatIsAskedOfIt(t *testing.T) {
	at := func(baseURL, name string) string {
		return embed.ServiceModel{BaseURL: baseURL, Name: name}.GetAddress()
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
	named := embed.LocalModel{Name: repository, File: embed.ModelFile}.GetAddress()
	if got := (embed.LocalModel{Name: repository}).GetAddress(); got != named {
		t.Errorf("%q and %q", got, named)
	}
	// Another build of one repository is another set of vectors.
	if got := (embed.LocalModel{Name: repository, File: "model_int8.onnx"}).GetAddress(); got == named {
		t.Errorf("two builds are both %q", got)
	}
	// A folder is what the weights are read from, and is written how it is
	// written.
	held := embed.LocalModel{Dir: "/models/e5"}.GetAddress()
	if got := (embed.LocalModel{Dir: "/models/e5/"}).GetAddress(); got != held {
		t.Errorf("%q and %q", got, held)
	}
	if held == named {
		t.Errorf("a folder and a repository are both %q", held)
	}
}

// A provider naming neither is an installation with no model, and has no
// address at all.
func TestAProviderThatNamesNeitherIsNowhere(t *testing.T) {
	if got := (embed.Provider{}).GetAddress(); got != "" {
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

// The engine is the settings' where they name one. A model is run the same way
// whatever else the file says about it.
func TestTheEngineNamedIsTheEngineRun(t *testing.T) {
	for _, named := range []string{embed.EngineRuntime, embed.EnginePureGo} {
		if got := (embed.LocalModel{Engine: named}).GetEngine(); got != named {
			t.Errorf("named %q, run on %q", named, got)
		}
	}
}

// A machine the runtime is published for runs the model through it, and one it
// is not published for runs the model on what is in the binary. Either way the
// settings name nothing.
func TestAnEngineNobodyNamedIsTheOneThisPlatformHas(t *testing.T) {
	want := embed.EnginePureGo
	if onnxruntime.IsPublished() {
		want = embed.EngineRuntime
	}
	if got := (embed.LocalModel{}).GetEngine(); got != want {
		t.Errorf("got %q, want %q on %s/%s", got, want, runtime.GOOS, runtime.GOARCH)
	}
}

// A person moving a vault between machines carries the file, and the engine
// written on one of them is read on the other.
func TestTheEngineIsReadOutOfTheFile(t *testing.T) {
	var held embed.LocalModel
	if err := json.Unmarshal([]byte(`{"engine":"go","runtime":"/opt/lib/libonnxruntime.so","threads":2}`), &held); err != nil {
		t.Fatal(err)
	}
	if held.GetEngine() != embed.EnginePureGo {
		t.Errorf("run on %q", held.GetEngine())
	}
	if held.Runtime != "/opt/lib/libonnxruntime.so" {
		t.Errorf("the runtime is %q", held.Runtime)
	}
	if held.GetThreads() != 2 {
		t.Errorf("one pass takes %d threads", held.GetThreads())
	}
}
