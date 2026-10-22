package container

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"

	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
)

func TestAProvisioningIsATaskUntilItIsReady(t *testing.T) {
	host := plugin.NewHost("desktop.editor")
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), 0) })
	registry := newDependencies()
	host.SetDependencies(registry)
	tasks := run.NewTasks()
	Config{PluginHost: host}.ReportDependencies(tasks)
	req := pluginport.DependencyRequirement{ID: "whisper", Kind: pluginport.DependencyKindModel}

	registry.Observe(req, pluginport.DependencyStatus{State: pluginport.DependencyStateProvisioning, BytesDownloaded: 5, BytesTotal: 10})
	list := tasks.List()
	if len(list) != 1 || list[0].ID != "provision:whisper" || list[0].Title != "Downloading whisper" ||
		list[0].Unit != run.Bytes || list[0].Count != 5 || list[0].Total != 10 || list[0].IsUserStarted {
		t.Fatalf("tasks = %+v", list)
	}
	registry.Observe(req, pluginport.DependencyStatus{State: pluginport.DependencyStateFailed, Error: "boom"})
	if list = tasks.List(); len(list) != 1 || list[0].Error != "boom" {
		t.Fatalf("a failure left tasks %+v", list)
	}
	registry.Observe(req, pluginport.DependencyStatus{State: pluginport.DependencyStateReady})
	if list = tasks.List(); len(list) != 0 {
		t.Errorf("a ready dependency left tasks %+v", list)
	}
}
