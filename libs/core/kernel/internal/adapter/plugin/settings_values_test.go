package plugin

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

const linkedPlugin = "md.numen.sample"

func TestReplacingASettingReturnsTheValueItReplacedAndWhetherTheDependenciesChanged(t *testing.T) {
	host := newLinkedHost(t)

	previous, hadPrevious, changesDependencies := host.Settings.ReplaceSetting(linkedPlugin, "model", "small")
	if hadPrevious || previous != nil || !changesDependencies {
		t.Errorf("first replace = %v, %v, %v, want no previous value and new dependencies", previous, hadPrevious, changesDependencies)
	}

	previous, hadPrevious, changesDependencies = host.Settings.ReplaceSetting(linkedPlugin, "model", "small")
	if previous != "small" || !hadPrevious || changesDependencies {
		t.Errorf("second replace = %v, %v, %v, want the stored value and the same dependencies", previous, hadPrevious, changesDependencies)
	}

	_, _, changesDependencies = host.Settings.ReplaceSetting(linkedPlugin, "dictionary", "words.txt")
	if changesDependencies {
		t.Error("a setting that selects no dependency changed the dependencies")
	}
}

func TestRestoreSettingReplacesTheValueOrRemovesTheSetting(t *testing.T) {
	host := newLinkedHost(t)
	host.Settings.SetSettings(map[string]map[string]any{linkedPlugin: {"model": "small"}})

	previous, hadPrevious, _ := host.Settings.ReplaceSetting(linkedPlugin, "model", "large")
	host.Settings.RestoreSetting(linkedPlugin, "model", previous, hadPrevious)
	if got := host.Settings.GetPluginSettings(linkedPlugin)["model"]; got != "small" {
		t.Errorf("model = %v after the restore, want small", got)
	}

	previous, hadPrevious, _ = host.Settings.ReplaceSetting(linkedPlugin, "dictionary", "words.txt")
	host.Settings.RestoreSetting(linkedPlugin, "dictionary", previous, hadPrevious)
	if _, stored := host.Settings.GetPluginSettings(linkedPlugin)["dictionary"]; stored {
		t.Error("a setting that had no value is still stored after the restore")
	}
}

func TestAFailedStartForAListenerIsReportedOnlyOnce(t *testing.T) {
	host := NewHost(testsupport.GetMachine(), "desktop.editor")
	failure := errors.New("exec format error")

	if host.Failures.MarkTargetResult("md.numen.sample", failure) {
		t.Error("the first failure was already reported")
	}
	if !host.Failures.MarkTargetResult("md.numen.sample", failure) {
		t.Error("the second failure in a row was not recognized as reported")
	}
	if !host.Failures.MarkTargetResult("md.numen.sample", nil) {
		t.Error("a start that succeeded did not see the failure before it")
	}
	if host.Failures.MarkTargetResult("md.numen.sample", failure) {
		t.Error("a failure after a success was counted as reported")
	}
}
