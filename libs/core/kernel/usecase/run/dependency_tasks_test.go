package run

import (
	"testing"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
)

func TestAProvisioningIsATaskUntilItIsReady(t *testing.T) {
	tasks := NewTasks()
	show := NewDependencyTasks(tasks)

	show.Execute("whisper", plugindomain.DependencyStatus{State: plugindomain.DependencyStateProvisioning, BytesDownloaded: 5, BytesTotal: 10})
	list := tasks.List()
	if len(list) != 1 || list[0].ID != "provision:whisper" || list[0].Title != "Downloading whisper" ||
		list[0].Unit != work.Bytes || list[0].Count != 5 || list[0].Total != 10 || list[0].IsUserStarted {
		t.Fatalf("tasks = %+v", list)
	}
	show.Execute("whisper", plugindomain.DependencyStatus{State: plugindomain.DependencyStateFailed, Error: "boom"})
	if list = tasks.List(); len(list) != 1 || list[0].Error != "boom" {
		t.Fatalf("a failure left tasks %+v", list)
	}
	show.Execute("whisper", plugindomain.DependencyStatus{State: plugindomain.DependencyStateReady})
	if list = tasks.List(); len(list) != 0 {
		t.Errorf("a ready dependency left tasks %+v", list)
	}
}

func TestAStoppedPluginIsATaskUntilItsFailureIsCleared(t *testing.T) {
	tasks := NewTasks()
	show := NewPluginFailureTasks(tasks)

	show.Report(plugindomain.Failure{PluginID: "ocr", Name: "Text recognition", Reason: "exited"})
	list := tasks.List()
	if len(list) != 1 || list[0].ID != "plugin-failed:ocr" || list[0].Title != "Plugin stopped" || list[0].Error != "exited" || list[0].PluginName != "Text recognition" {
		t.Fatalf("tasks = %+v", list)
	}
	show.Clear("ocr")
	if list = tasks.List(); len(list) != 0 {
		t.Errorf("a cleared failure left tasks %+v", list)
	}
}
