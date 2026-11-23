package supervision

import (
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/errsink"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

type failureClock struct{}

func (failureClock) Now() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }

type failureHealth struct{}

func (failureHealth) Set(plugin.HealthItem) {}
func (failureHealth) Remove(string, string) {}

func TestRecordedActivationFailureNotifiesReporter(t *testing.T) {
	sups := NewSupervisors(func(error) {}, func(string) {})
	sup := sups.Register(manifest.DiscoveredPlugin{Manifest: manifest.Manifest{ID: "example.plugin"}})
	failures := NewFailures(sups, failureClock{}, &errsink.Sink{}, failureHealth{})
	var reported []plugin.Failure
	failures.SetOnFailure(func(failure plugin.Failure) { reported = append(reported, failure) })
	sup.SetActivating(plugin.Holder{})
	failures.FailActivation(t.Context(), "example.plugin", "Example", sup, errors.New("startup failed"))
	if len(reported) != 1 || reported[0].PluginID != "example.plugin" || reported[0].Detail != "startup failed" {
		t.Fatalf("reported = %+v", reported)
	}
}
