package proofreading_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
)

// read is the configuration a file holds.
func read(t *testing.T, raw string) proofreading.Config {
	t.Helper()
	var cfg proofreading.Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

// profile is the one profile a configuration carries under a name.
func profile(t *testing.T, cfg proofreading.Config, name string) proofreading.Profile {
	t.Helper()
	held, carried := cfg.Profiles[name]
	if !carried {
		t.Fatalf("no profile named %q among %v", name, cfg.Profiles)
	}
	return held
}

// A profile keeps the defaults of the kind it says it is for the fields the
// file leaves out.
func TestAProfileKeepsTheDefaultsOfItsKind(t *testing.T) {
	cfg := read(t, `{"profiles":{
		"openrouter":{"use":"service","name":"some-model"},
		"mine":{"use":"agent","model":"haiku"}
	}}`)

	service := profile(t, cfg, "openrouter")
	if service.BaseURL != proofreading.ServiceDefaults().BaseURL {
		t.Errorf("a service points at %q", service.BaseURL)
	}
	if service.Attempts != proofreading.DefaultAttempts || service.DelayMS != proofreading.DefaultDelayMS ||
		service.MaxDelayMS != proofreading.DefaultMaxDelayMS {
		t.Errorf("a service retries %d times from %d to %d", service.Attempts, service.DelayMS, service.MaxDelayMS)
	}
	if service.BatchSize != proofreading.DefaultBatchSize || service.InFlight != 0 {
		t.Errorf("a service carries %d lines and %d at once, and names no number at once", service.BatchSize, service.InFlight)
	}

	agent := profile(t, cfg, "mine")
	if agent.BatchSize != proofreading.DefaultAgentBatchSize ||
		agent.InFlight != 0 {
		t.Errorf("the command line carries %d lines and %d at once", agent.BatchSize, agent.InFlight)
	}
	if agent.BaseURL != "" {
		t.Errorf("the command line points at %q", agent.BaseURL)
	}
}

// One threshold stands over every profile, and a file naming none takes what
// was measured.
func TestOneThresholdStandsOverEveryProfile(t *testing.T) {
	if got := read(t, `{}`).Distance(); got != proofreading.DefaultMaxEditDistance {
		t.Errorf("a file naming nothing holds a correction to %v", got)
	}
	if got := read(t, `{"max_edit_distance":0.1}`).Distance(); got != 0.1 {
		t.Errorf("a file naming 0.1 holds a correction to %v", got)
	}
	if read(t, `{}`).HasProfiles() {
		t.Error("a file naming no profile was said to name one")
	}
}

// A key is written by a person and never by us: no value that formats or
// serialises itself carries it back out.
func TestAKeyIsNeverWrittenBackOut(t *testing.T) {
	const secret = "sk-a-key-a-person-wrote"
	cfg := read(t, `{"profiles":{"openrouter":{
		"use":"service","name":"some-model","key":"`+secret+`"}}}`)

	held := profile(t, cfg, "openrouter")
	if held.Key() != secret {
		t.Fatalf("the key the file names was read as %q", held.Key())
	}

	written, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(written), secret) {
		t.Errorf("the key was written back out:\n%s", written)
	}
	if strings.Contains(held.String(), secret) {
		t.Errorf("the profile reports itself as %q", held)
	}
}

// What is written is read back as the same profile, so a file this application
// rewrites says what it said.
func TestAProfileIsReadBackAsWhatItWas(t *testing.T) {
	for _, one := range []struct {
		what string
		raw  string
	}{
		{"a service", `{"use":"service","base_url":"https://elsewhere/v1",
			"batch_url":"https://elsewhere/batches","name":"some-model",
			"key_env":"SOMEWHERE_KEY","batch_size":12,"overlap":3,"in_flight":7,
			"attempts":2,"delay_ms":250,"max_delay_ms":9000}`},
		{"the command line", `{"use":"agent","model":"haiku",
			"command":["/somewhere/claude","--flag"],"batch_size":80,"overlap":4,"in_flight":1}`},
	} {
		t.Run(one.what, func(t *testing.T) {
			var was proofreading.Profile
			if err := json.Unmarshal([]byte(one.raw), &was); err != nil {
				t.Fatal(err)
			}
			written, err := json.Marshal(was)
			if err != nil {
				t.Fatal(err)
			}
			var again proofreading.Profile
			if err := json.Unmarshal(written, &again); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(again, was) {
				t.Errorf("read back as\n%+v\nand not\n%+v\nfrom %s", again, was, written)
			}
		})
	}
}

// A profile says which model corrects a reading, by the name the thing reaching
// it knows a model under.
func TestAProfileWithNoModelNamesNone(t *testing.T) {
	for _, one := range []struct {
		what    string
		raw     string
		isNamed bool
	}{
		{"a profile the file leaves empty", `{}`, false},
		{"a service naming no model", `{"use":"service"}`, false},
		{"a service naming one", `{"use":"service","name":"some-model"}`, true},
		{"the command line naming no model", `{"use":"agent"}`, false},
		{"the command line naming one", `{"use":"agent","model":"haiku"}`, true},
		{"a use nothing here reaches", `{"use":"telepathy","name":"some-model"}`, false},
	} {
		t.Run(one.what, func(t *testing.T) {
			var held proofreading.Profile
			if err := json.Unmarshal([]byte(one.raw), &held); err != nil {
				t.Fatal(err)
			}
			if held.HasModel() != one.isNamed {
				t.Errorf("%s names a model: %v", one.raw, held.HasModel())
			}
		})
	}
}

// A key the file does not carry is read from the environment, from the variable
// the profile names or the one it does not have to.
func TestAKeyTheFileDoesNotCarryComesFromTheEnvironment(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "sk-from-the-usual-place")
	t.Setenv("SOMEWHERE_KEY", "sk-from-somewhere")

	var usual, named proofreading.Profile
	if err := json.Unmarshal([]byte(`{"use":"service"}`), &usual); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"use":"service","key_env":"SOMEWHERE_KEY"}`), &named); err != nil {
		t.Fatal(err)
	}
	if usual.Key() != "sk-from-the-usual-place" {
		t.Errorf("the usual place holds %q", usual.Key())
	}
	if named.Key() != "sk-from-somewhere" {
		t.Errorf("the variable the profile names holds %q", named.Key())
	}
}

func TestAProfileNamingNoNumberAtOnceTakesTheNumberPlannedForIt(t *testing.T) {
	cfg := read(t, `{"profiles":{
		"openrouter":{"use":"service","name":"some-model"},
		"mine":{"use":"agent","model":"haiku"}
	}}`)
	for _, name := range []string{"openrouter", "mine"} {
		if got := profile(t, cfg, name).GetInFlight(7); got != 7 {
			t.Errorf("%s asks about %d batches at once, want the 7 planned", name, got)
		}
	}
}

func TestAProfileNamingANumberAtOnceKeepsItWhateverIsPlanned(t *testing.T) {
	cfg := read(t, `{"profiles":{"openrouter":{"use":"service","name":"some-model","in_flight":3}}}`)
	if got := profile(t, cfg, "openrouter").GetInFlight(7); got != 3 {
		t.Errorf("a service asks about %d batches at once, want the 3 it names", got)
	}
}

func TestAProfileWithNothingPlannedTakesTheDefaultOfItsKind(t *testing.T) {
	cfg := read(t, `{"profiles":{
		"openrouter":{"use":"service","name":"some-model"},
		"mine":{"use":"agent","model":"haiku"}
	}}`)
	if got := profile(t, cfg, "openrouter").GetInFlight(0); got != proofreading.DefaultInFlight {
		t.Errorf("a service asks about %d batches at once, want %d", got, proofreading.DefaultInFlight)
	}
	if got := profile(t, cfg, "mine").GetInFlight(0); got != proofreading.DefaultAgentInFlight {
		t.Errorf("the command line asks about %d batches at once, want %d", got, proofreading.DefaultAgentInFlight)
	}
}

func TestANumberAtOnceIsWrittenOnlyWhenTheProfileNamesOne(t *testing.T) {
	cfg := read(t, `{"profiles":{
		"unnamed":{"use":"service","name":"some-model"},
		"named":{"use":"service","name":"some-model","in_flight":3}
	}}`)
	if raw, _ := json.Marshal(profile(t, cfg, "unnamed")); strings.Contains(string(raw), "in_flight") {
		t.Errorf("a profile naming no number is written as %s", raw)
	}
	if raw, _ := json.Marshal(profile(t, cfg, "named")); !strings.Contains(string(raw), `"in_flight":3`) {
		t.Errorf("a profile naming 3 is written as %s", raw)
	}
}
