package run

import (
	"testing"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
)

func TestAProvisioningIsATaskUntilItIsReady(t *testing.T) {
	tasks := NewTasks()
	show := NewDependencyTasks(tasks)

	show.Execute("whisper", "whisper", plugindomain.DependencyStatus{State: plugindomain.DependencyStateProvisioning, BytesDownloaded: 5, BytesTotal: 10}, nil)
	list := tasks.List()
	if len(list) != 1 || list[0].ID != "provision:whisper" || list[0].Title != "Downloading whisper" ||
		list[0].Unit != work.Bytes || list[0].Count != 5 || list[0].Total != 10 || list[0].IsUserStarted {
		t.Fatalf("tasks = %+v", list)
	}
	show.Execute("whisper", "whisper", plugindomain.DependencyStatus{State: plugindomain.DependencyStateFailed, Error: "boom"}, nil)
	if list = tasks.List(); len(list) != 1 || list[0].Error != "boom" {
		t.Fatalf("a failure left tasks %+v", list)
	}
	show.Execute("whisper", "whisper", plugindomain.DependencyStatus{State: plugindomain.DependencyStateReady}, nil)
	if list = tasks.List(); len(list) != 0 {
		t.Errorf("a ready dependency left tasks %+v", list)
	}
}

func TestCancelingATaskOfADownloadStopsThatDownload(t *testing.T) {
	tasks := NewTasks()
	show := NewDependencyTasks(tasks)
	canceled := 0

	show.Execute("whisper", "whisper", plugindomain.DependencyStatus{State: plugindomain.DependencyStateProvisioning}, func() { canceled++ })
	if !tasks.Cancel("provision:whisper") || canceled != 1 {
		t.Fatalf("Cancel = stopped %d downloads, want 1", canceled)
	}
	show.Execute("whisper", "whisper", plugindomain.DependencyStatus{State: plugindomain.DependencyStateMissing}, nil)
	if list := tasks.List(); len(list) != 0 {
		t.Errorf("a canceled download left tasks %+v", list)
	}
}

func TestADownloadTaskShowsTheNameOfTheDependency(t *testing.T) {
	tasks := NewTasks()
	NewDependencyTasks(tasks).Execute("0a1b", "latin_rec model", plugindomain.DependencyStatus{State: plugindomain.DependencyStateProvisioning}, nil)
	list := tasks.List()
	if len(list) != 1 || list[0].ID != "provision:0a1b" || list[0].Title != "Downloading latin_rec model" {
		t.Fatalf("tasks = %+v", list)
	}
}
