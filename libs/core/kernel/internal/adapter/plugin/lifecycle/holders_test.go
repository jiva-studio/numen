package lifecycle

import (
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

var (
	useLease      = Lease{Kind: UseLease, Holder: plugin.Holder{Who: plugin.WhoWindow, Why: "open the note"}}
	questionLease = Lease{Kind: QuestionLease, Holder: plugin.Holder{Who: plugin.WhoSettings, Why: "list tools"}}
)

func TestTheLeasesOfAPluginNameTheirHolders(t *testing.T) {
	starter := Lease{Kind: UseLease, Holder: plugin.Holder{Who: plugin.WhoIndexing, Why: "embed notes"}}
	asker := Lease{Kind: UseLease, Holder: plugin.Holder{PluginID: "org.example.orchard", Why: "call tool"}}
	sup := New(nil)
	start := testsupport.GetEpoch()
	sup.SetActivating()
	sup.SetStartedBy(starter.Holder)
	sup.ActivateInProcess(starter, start)
	sup.Acquire(asker, start.Add(time.Second))
	sup.Acquire(questionLease, start.Add(2*time.Second))

	usage, _ := sup.GetUsage()
	if usage.StartedBy != starter.Holder {
		t.Errorf("started by %+v, want %+v", usage.StartedBy, starter.Holder)
	}
	if want := []plugin.Holder{starter.Holder, asker.Holder, questionLease.Holder}; !slices.Equal(usage.Holders, want) {
		t.Errorf("holders %+v, want %+v", usage.Holders, want)
	}

	sup.Release(asker, start.Add(3*time.Second))

	usage, _ = sup.GetUsage()
	if want := []plugin.Holder{starter.Holder, questionLease.Holder}; !slices.Equal(usage.Holders, want) {
		t.Errorf("holders after the release %+v, want %+v", usage.Holders, want)
	}
	if usage.StartedBy != starter.Holder {
		t.Errorf("after the release started by %+v, want %+v", usage.StartedBy, starter.Holder)
	}
}

func TestAReleaseOfTheSameHolderTwiceRemovesOneLeaseEachTime(t *testing.T) {
	sup := New(nil)
	start := testsupport.GetEpoch()
	sup.SetActivating()
	sup.ActivateInProcess(useLease, start)
	sup.Acquire(useLease, start)
	sup.Acquire(questionLease, start)

	sup.Release(useLease, start)

	usage, _ := sup.GetUsage()
	if want := []plugin.Holder{useLease.Holder, questionLease.Holder}; !slices.Equal(usage.Holders, want) {
		t.Errorf("holders %+v, want %+v", usage.Holders, want)
	}
}

func TestAStoppedPluginHasNoHolders(t *testing.T) {
	sup := New(nil)
	start := testsupport.GetEpoch()
	sup.SetActivating()
	sup.ActivateInProcess(useLease, start)
	sup.Acquire(questionLease, start)

	sup.SetDormant(nil)
	sup.SetActivating()
	sup.ActivateInProcess(questionLease, start)

	usage, _ := sup.GetUsage()
	if want := []plugin.Holder{questionLease.Holder}; !slices.Equal(usage.Holders, want) {
		t.Errorf("holders %+v, want %+v", usage.Holders, want)
	}
}
