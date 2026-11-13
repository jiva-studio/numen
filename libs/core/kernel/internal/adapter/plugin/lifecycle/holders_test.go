package lifecycle

import (
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func getUseLease() Lease {
	return Lease{Kind: UseLease, Holder: plugin.Holder{Who: plugin.WhoWindow, Why: "open the note"}}
}

func getQuestionLease() Lease {
	return Lease{Kind: QuestionLease, Holder: plugin.Holder{Who: plugin.WhoSettings, Why: "list tools"}}
}

func TestTheLeasesOfAPluginNameTheirHolders(t *testing.T) {
	starter := Lease{Kind: UseLease, Holder: plugin.Holder{Who: plugin.WhoIndexing, Why: "embed notes"}}
	asker := Lease{Kind: UseLease, Holder: plugin.Holder{PluginID: "org.example.orchard", Why: "call tool"}}
	sup := New(nil)
	start := testsupport.GetEpoch()
	sup.SetActivating(starter.Holder)
	sup.ActivateInProcess(starter, start)
	sup.Acquire(asker, start.Add(time.Second))
	sup.Acquire(getQuestionLease(), start.Add(2*time.Second))

	usage, _ := sup.GetUsage()
	if usage.StartedBy != starter.Holder {
		t.Errorf("started by %+v, want %+v", usage.StartedBy, starter.Holder)
	}
	if want := []plugin.Holder{starter.Holder, asker.Holder, getQuestionLease().Holder}; !slices.Equal(usage.Holders, want) {
		t.Errorf("holders %+v, want %+v", usage.Holders, want)
	}

	sup.Release(asker, start.Add(3*time.Second))

	usage, _ = sup.GetUsage()
	if want := []plugin.Holder{starter.Holder, getQuestionLease().Holder}; !slices.Equal(usage.Holders, want) {
		t.Errorf("holders after the release %+v, want %+v", usage.Holders, want)
	}
	if usage.StartedBy != starter.Holder {
		t.Errorf("after the release started by %+v, want %+v", usage.StartedBy, starter.Holder)
	}
}

func TestAReleaseOfTheSameHolderTwiceRemovesOneLeaseEachTime(t *testing.T) {
	sup := New(nil)
	start := testsupport.GetEpoch()
	sup.SetActivating(plugin.Holder{})
	sup.ActivateInProcess(getUseLease(), start)
	sup.Acquire(getUseLease(), start)
	sup.Acquire(getQuestionLease(), start)

	sup.Release(getUseLease(), start)

	usage, _ := sup.GetUsage()
	if want := []plugin.Holder{getUseLease().Holder, getQuestionLease().Holder}; !slices.Equal(usage.Holders, want) {
		t.Errorf("holders %+v, want %+v", usage.Holders, want)
	}
}

func TestAStoppedPluginHasNoHolders(t *testing.T) {
	sup := New(nil)
	start := testsupport.GetEpoch()
	sup.SetActivating(plugin.Holder{})
	sup.ActivateInProcess(getUseLease(), start)
	sup.Acquire(getQuestionLease(), start)

	sup.SetDormant(nil)
	sup.SetActivating(plugin.Holder{})
	sup.ActivateInProcess(getQuestionLease(), start)

	usage, _ := sup.GetUsage()
	if want := []plugin.Holder{getQuestionLease().Holder}; !slices.Equal(usage.Holders, want) {
		t.Errorf("holders %+v, want %+v", usage.Holders, want)
	}
}
