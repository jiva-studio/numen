package besthealth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/besthealth"
)

type refusing struct{}

func (refusing) Report(context.Context, health.Item) error { return errors.New("invalid_argument") }
func (refusing) Clear(context.Context, string) error       { return errors.New("invalid_argument") }

func TestRefusedReportsAreLoggedAndNeverReturned(t *testing.T) {
	var warned []string
	reports := besthealth.New(refusing{}, func(message string, _ error) { warned = append(warned, message) })
	if err := reports.Report(t.Context(), health.Item{Key: "Bad/Key", Severity: health.Warning, Text: "x"}); err != nil {
		t.Fatalf("Report returned %v", err)
	}
	if err := reports.Clear(t.Context(), "Bad/Key"); err != nil {
		t.Fatalf("Clear returned %v", err)
	}
	if len(warned) != 2 {
		t.Fatalf("warned %d times, want 2", len(warned))
	}
}
