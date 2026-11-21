package run

import (
	"testing"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestAProvisioningIsATaskUntilItIsReady(t *testing.T) {
	tasks := NewTasks()
	show := NewDependencyTasks(tasks)
	whisper := manifest.DependencyRequirement{ID: "whisper"}

	show.Execute(whisper, plugindomain.DependencyStatus{State: plugindomain.DependencyStateProvisioning, BytesDownloaded: 5, BytesTotal: 10}, nil)
	list := tasks.List()
	if len(list) != 1 || list[0].ID != "provision:whisper" || list[0].Title != "Downloading" || list[0].About != "whisper" ||
		list[0].Unit != work.Bytes || list[0].Count != 5 || list[0].Total != 10 || list[0].IsUserStarted {
		t.Fatalf("tasks = %+v", list)
	}
	show.Execute(whisper, plugindomain.DependencyStatus{State: plugindomain.DependencyStateFailed, Error: "boom"}, nil)
	if list = tasks.List(); len(list) != 1 || list[0].Error != "boom" {
		t.Fatalf("a failure left tasks %+v", list)
	}
	show.Execute(whisper, plugindomain.DependencyStatus{State: plugindomain.DependencyStateReady}, nil)
	if list = tasks.List(); len(list) != 0 {
		t.Errorf("a ready dependency left tasks %+v", list)
	}
}

func TestCancelingATaskOfADownloadStopsThatDownload(t *testing.T) {
	tasks := NewTasks()
	show := NewDependencyTasks(tasks)
	whisper := manifest.DependencyRequirement{ID: "whisper"}
	canceled := 0

	show.Execute(whisper, plugindomain.DependencyStatus{State: plugindomain.DependencyStateProvisioning}, func() { canceled++ })
	if !tasks.Cancel("provision:whisper") || canceled != 1 {
		t.Fatalf("Cancel = stopped %d downloads, want 1", canceled)
	}
	show.Execute(whisper, plugindomain.DependencyStatus{State: plugindomain.DependencyStateMissing}, nil)
	if list := tasks.List(); len(list) != 0 {
		t.Errorf("a canceled download left tasks %+v", list)
	}
}

func TestADownloadIsNamedByItsFileNotItsHash(t *testing.T) {
	tasks := NewTasks()
	show := NewDependencyTasks(tasks)
	hash := "b6deea7f2e22c10c043019f294a0ea4d2a6c0ae52a009c34847640db75ec5580"
	req := manifest.DependencyRequirement{ID: hash, SourceURL: "https://example.com/models/ggml-base.bin"}

	show.Execute(req, plugindomain.DependencyStatus{State: plugindomain.DependencyStateProvisioning}, nil)
	if list := tasks.List(); len(list) != 1 || list[0].About != "ggml-base.bin" {
		t.Fatalf("tasks = %+v", list)
	}
}
