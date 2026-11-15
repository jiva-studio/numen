package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/plugins/tools-updater/backend/api"
)

func startTestCore(t *testing.T) *linkedCore {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_CACHE_HOME"} {
		t.Setenv(name, dir)
	}
	var errOut bytes.Buffer
	core, err := startCore(dir, &errOut)
	if err != nil {
		t.Fatalf("startCore: %v\n%s", err, errOut.String())
	}
	t.Cleanup(func() {
		if err := core.Close(); err != nil {
			t.Error(err)
		}
	})
	return core
}

func getUpdateStatus(t *testing.T, core *linkedCore) api.Status {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, core.url+"/"+api.Service+"/"+api.MethodGetStatus, bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+core.token)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := res.Body.Close(); err != nil {
			t.Error(err)
		}
	})
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("the plugin answered %s: %s", res.Status, body)
	}
	var status api.Status
	if err := json.Unmarshal(body, &status); err != nil {
		t.Fatal(err)
	}
	return status
}

// A plugin linked into the binary answers a window on the route a plugin process would answer on.
func TestLinkedPluginAnswersOnTheRouteOfTheWindow(t *testing.T) {
	core := startTestCore(t)

	if status := getUpdateStatus(t, core); status.State != api.StateUnchecked || status.Update != nil {
		t.Errorf("the plugin answered %+v, want an unchecked state", status)
	}
}

// The process list the window reads reports the plugin that answered as running in the process of the core.
func TestLinkedPluginIsListedAsRunningInProcess(t *testing.T) {
	core := startTestCore(t)
	getUpdateStatus(t, core)

	client := numenv1connect.NewPluginServiceClient(http.DefaultClient, core.url)
	req := connect.NewRequest(&v1.WatchPluginProcessesRequest{})
	req.Header().Set("Authorization", "Bearer "+core.token)
	stream, err := client.WatchPluginProcesses(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := stream.Close(); err != nil {
			t.Error(err)
		}
	})
	if !stream.Receive() {
		t.Fatalf("no list of the plugins that run: %v", stream.Err())
	}
	processes := stream.Msg().GetProcesses()
	if len(processes) != 1 || !processes[0].GetIsInProcess() {
		t.Errorf("the plugins that run are %v, want the linked plugin in process", processes)
	}
}
