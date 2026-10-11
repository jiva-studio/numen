package proofreading_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
)

// What a file naming nothing but the kind of each profile is read as.
func TestAFileNamingNothingTakesTheDecidedDefaults(t *testing.T) {
	cfg := read(t, `{"profiles":{"openrouter":{"use":"service"},"mine":{"use":"agent"}}}`)
	if cfg.MaxEditDistance != 0.30 {
		t.Errorf("a correction is held to %v", cfg.MaxEditDistance)
	}

	service := profile(t, cfg, "openrouter")
	if service.BatchSize != 40 || service.Attempts != 4 || service.DelayMS != 500 || service.MaxDelayMS != 30_000 {
		t.Errorf("a service carries %d lines, is sent %d times, waits from %d to %d ms",
			service.BatchSize, service.Attempts, service.DelayMS, service.MaxDelayMS)
	}
	if got := service.GetInFlight(0); got != 4 {
		t.Errorf("a service is asked about %d batches at once", got)
	}

	agent := profile(t, cfg, "mine")
	if agent.BatchSize != 60 {
		t.Errorf("the command line carries %d lines", agent.BatchSize)
	}
	if got := agent.GetInFlight(0); got != 2 {
		t.Errorf("the command line is asked about %d batches at once", got)
	}
}

func TestTheDefaultsHoldACorrectionToTheMeasuredDistance(t *testing.T) {
	if got := proofreading.Defaults().MaxEditDistance; got != proofreading.DefaultMaxEditDistance {
		t.Errorf("the defaults hold a correction to %v", got)
	}
}

func TestAConfigurationNamingNoDistanceTakesTheMeasuredOne(t *testing.T) {
	if got := (proofreading.Config{}).Distance(); got != proofreading.DefaultMaxEditDistance {
		t.Errorf("a configuration naming nothing holds a correction to %v", got)
	}
}

// Reading a file replaces what the configuration held before.
func TestAFileReadOverAConfigurationStartsFromTheDefaults(t *testing.T) {
	cfg := proofreading.Config{MaxEditDistance: 0.9}
	if err := json.Unmarshal([]byte(`{}`), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.MaxEditDistance != proofreading.DefaultMaxEditDistance {
		t.Errorf("a file naming nothing holds a correction to %v", cfg.MaxEditDistance)
	}
}

func TestOneProfileIsProfiles(t *testing.T) {
	if !read(t, `{"profiles":{"openrouter":{"use":"service"}}}`).HasProfiles() {
		t.Error("a file naming one profile was said to name none")
	}
}

func TestAFieldOfTheWrongTypeIsRefused(t *testing.T) {
	for _, raw := range []string{
		`{"max_edit_distance":"high"}`,
		`{"profiles":{"openrouter":{"use":"service","batch_size":"many"}}}`,
	} {
		var cfg proofreading.Config
		if err := json.Unmarshal([]byte(raw), &cfg); err == nil {
			t.Errorf("%s was read as %+v", raw, cfg)
		}
	}
}

func TestEveryFieldAFileNamesIsRead(t *testing.T) {
	var service proofreading.Profile
	if err := json.Unmarshal([]byte(`{"use":"service","base_url":"https://elsewhere/v1",
		"batch_url":"https://elsewhere/batches","batch_size":12,"overlap":3,
		"attempts":2,"delay_ms":250,"max_delay_ms":9000}`), &service); err != nil {
		t.Fatal(err)
	}
	if service.BaseURL != "https://elsewhere/v1" || service.BatchURL != "https://elsewhere/batches" {
		t.Errorf("a service points at %q and %q", service.BaseURL, service.BatchURL)
	}
	if service.BatchSize != 12 || service.Overlap != 3 {
		t.Errorf("a service carries %d lines, %d shared", service.BatchSize, service.Overlap)
	}
	if service.Attempts != 2 || service.DelayMS != 250 || service.MaxDelayMS != 9000 {
		t.Errorf("a service is sent %d times, waiting from %d to %d ms", service.Attempts, service.DelayMS, service.MaxDelayMS)
	}

	var agent proofreading.Profile
	if err := json.Unmarshal([]byte(`{"use":"agent","command":["/somewhere/claude"]}`), &agent); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(agent.Command, []string{"/somewhere/claude"}) {
		t.Errorf("the command line is started as %q", agent.Command)
	}
}

// A profile of a kind nothing here reaches keeps that kind, and nothing it held
// before.
func TestAProfileOfAnUnknownKindKeepsOnlyItsKind(t *testing.T) {
	held := proofreading.ServiceDefaults()
	if err := json.Unmarshal([]byte(`{"use":"telepathy"}`), &held); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(held, proofreading.Profile{Use: "telepathy"}) {
		t.Errorf("read as %+v", held)
	}
}

func TestACommandIsWrittenOnlyWhenTheProfileNamesOne(t *testing.T) {
	none, err := json.Marshal(proofreading.Profile{Use: proofreading.UseAgent, Model: "haiku"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(none), "command") {
		t.Errorf("a profile naming no command is written as %s", none)
	}
	one, err := json.Marshal(proofreading.Profile{Use: proofreading.UseAgent, Command: []string{"/somewhere/claude"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(one), `"command":["/somewhere/claude"]`) {
		t.Errorf("a profile naming one command is written as %s", one)
	}
}

func TestAProfileReportsItself(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "")
	for _, one := range []struct {
		held proofreading.Profile
		want string
	}{
		{proofreading.Profile{Use: proofreading.UseAgent, Model: "haiku"}, "the command line, model haiku"},
		{proofreading.Profile{Use: proofreading.UseService, Name: "some-model", BaseURL: "https://elsewhere/v1"},
			"service some-model at https://elsewhere/v1, key absent"},
	} {
		if got := one.held.String(); got != one.want {
			t.Errorf("reported as %q, want %q", got, one.want)
		}
	}

	t.Setenv(proofreading.KeyEnvVar, "sk-held")
	held := proofreading.Profile{Use: proofreading.UseService, Name: "some-model", BaseURL: "https://elsewhere/v1"}
	if got, want := held.String(), "service some-model at https://elsewhere/v1, key present"; got != want {
		t.Errorf("reported as %q, want %q", got, want)
	}
}

func TestAProfileNamingNoVariableReadsTheKeyFromTheUsualOne(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "sk-from-the-usual-place")
	if got := (proofreading.Profile{Use: proofreading.UseService}).Key(); got != "sk-from-the-usual-place" {
		t.Errorf("the key was read as %q", got)
	}
}

func TestOneBatchAtOnceIsANumber(t *testing.T) {
	if got := (proofreading.Profile{Use: proofreading.UseService, InFlight: 1}).GetInFlight(7); got != 1 {
		t.Errorf("a profile naming one is asked about %d at once", got)
	}
	if got := (proofreading.Profile{Use: proofreading.UseService}).GetInFlight(1); got != 1 {
		t.Errorf("a profile with one planned is asked about %d at once", got)
	}
}
