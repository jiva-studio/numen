package container_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
)

// A container nobody configured proofreads with nothing, and the machine's own
// settings are not read here.
func TestAContainerNobodyGaveSettingsProofreadsWithNothing(t *testing.T) {
	proofreader, why := container.Config{}.Proofreader()
	if why != nil {
		t.Fatalf("want no reason, got %v", why)
	}
	if proofreader != nil {
		t.Errorf("got a proofreader: %v", proofreader.Name())
	}
}

// A section that asks for a service without saying which model is a section
// that named nothing.
func TestAServiceWithoutAModelNameNamesNoProofreader(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "sk-test")
	cfg := proofreading.Defaults()
	cfg.Use = proofreading.UseService

	proofreader, why := container.Config{Proofreading: cfg}.Proofreader()
	if why != nil {
		t.Fatalf("want no reason, got %v", why)
	}
	if proofreader != nil {
		t.Errorf("got a proofreader: %v", proofreader.Name())
	}
}

func TestAModelWithoutAKeyGivesTheReasonAndNoProofreader(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "")
	cfg := proofreading.Defaults()
	cfg.Use = proofreading.UseService
	cfg.Service.Name = "some-model"

	proofreader, why := container.Config{Proofreading: cfg}.Proofreader()
	if why == nil {
		t.Fatal("no reason")
	}
	if !strings.Contains(why.Error(), "key") {
		t.Errorf("the reason does not say what is missing: %v", why)
	}
	if proofreader != nil {
		t.Errorf("got a proofreader: %v", proofreader.Name())
	}
}

func TestTheProofreadingSettingsGivenAreTheOnesUsed(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "sk-test")
	cfg := proofreading.Defaults()
	cfg.Use = proofreading.UseService
	cfg.Service.Name = "some-model"
	// Nowhere: no test reaches a service.
	cfg.Service.BaseURL = "http://127.0.0.1:1/v1"

	proofreader, why := container.Config{Proofreading: cfg}.Proofreader()
	if why != nil {
		t.Fatal(why)
	}
	if proofreader == nil {
		t.Fatal("no proofreader")
	}
	if got := proofreader.Name(); got != "some-model" {
		t.Errorf("got %q", got)
	}
}

// A proofreader configured with a queue is one the pages can be left with.
func TestAProofreaderWithAQueueLeavesPagesWithIt(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "sk-a-key")
	cfg := proofreading.Defaults()
	cfg.Use = proofreading.UseService
	cfg.Service.Name = "test-model"

	queue, why := container.Config{Proofreading: cfg}.ProofreadQueue()
	if why != nil {
		t.Fatalf("want no reason, got %v", why)
	}
	if queue == nil || queue.Name() != "test-model" {
		t.Fatalf("got %v", queue)
	}
}

// A service with nowhere to leave the pages is asked a page at a time.
func TestAServiceWithoutAQueueLeavesNothing(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "sk-a-key")
	cfg := proofreading.Defaults()
	cfg.Use = proofreading.UseService
	cfg.Service.Name = "test-model"
	cfg.Service.BatchURL = ""

	queue, why := container.Config{Proofreading: cfg}.ProofreadQueue()
	if why != nil {
		t.Fatalf("want no reason, got %v", why)
	}
	if queue != nil {
		t.Errorf("got a queue: %v", queue.Name())
	}
}
