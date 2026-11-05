package manifest_test

import (
	"errors"
	"slices"
	"testing"

	. "github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// newActivatedManifest is a valid plugin with a backend that declares the activation events.
func newActivatedManifest(events ...string) Manifest {
	return Manifest{
		ID: "org.example.sample", Name: "Sample", Version: "1.0.0",
		Backend: Backend{Entrypoint: "bin/sample"}, ActivationEvents: events,
	}
}

// hasActivationIssue reports whether the manifest breaks the activation event rule.
func hasActivationIssue(m Manifest) bool {
	return slices.ContainsFunc(m.ListIssues(), func(issue error) bool { return errors.Is(issue, ErrInvalidActivationEvent) })
}

func TestManifestWithWildcardActivationIsRejected(t *testing.T) {
	for _, events := range [][]string{{"*"}, {"onStartup", "*"}} {
		if !hasActivationIssue(newActivatedManifest(events...)) {
			t.Errorf("a manifest declaring %v is accepted", events)
		}
	}
}

func TestManifestAcceptsTheEventsTheHostRaises(t *testing.T) {
	for _, event := range []string{"onStartup", "onVectorsReady", "onFile:md", "onFile:pdf", "onFile:M4A"} {
		if hasActivationIssue(newActivatedManifest(event)) {
			t.Errorf("a manifest declaring %s is rejected", event)
		}
	}
}

func TestManifestRejectsAnEventTheHostDoesNotRaise(t *testing.T) {
	for _, event := range []string{"", "onCommand:run", "onstartup", "onFile:", "onFile:*", "onFile:notes/a.md", "onFile:.md", "onVectorsReady:01VAULT"} {
		if !hasActivationIssue(newActivatedManifest(event)) {
			t.Errorf("a manifest declaring %q is accepted", event)
		}
	}
}
