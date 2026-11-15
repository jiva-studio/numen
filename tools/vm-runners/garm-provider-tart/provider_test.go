package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const (
	testController = "controller-a"
	testPool       = "pool-a"
)

type fakeTart struct {
	vms         map[string]tartVM
	processes   map[int]string
	calls       []string
	killed      []int
	failOn      string
	isStopStuck bool
	nextPID     int
	detached    [][]string
}

func newFakeTart(vms ...tartVM) *fakeTart {
	f := &fakeTart{vms: map[string]tartVM{}, processes: map[int]string{}, nextPID: 100}
	for _, vm := range vms {
		f.vms[vm.Name] = vm
	}
	return f
}

func (f *fakeTart) runCommand(_ context.Context, name string, args ...string) ([]byte, error) {
	switch name {
	case "ps":
		pid, err := strconv.Atoi(args[2])
		if err != nil {
			return nil, err
		}
		line, ok := f.processes[pid]
		if !ok {
			return nil, errors.New("no such process")
		}
		return []byte(line + "\n"), nil
	case "kill":
		pid, err := strconv.Atoi(args[1])
		if err != nil {
			return nil, err
		}
		f.killed = append(f.killed, pid)
		delete(f.processes, pid)
		return nil, nil
	}
	f.calls = append(f.calls, strings.Join(args, " "))
	if f.failOn != "" && args[0] == f.failOn {
		return nil, fmt.Errorf("tart %s failed", args[0])
	}
	switch args[0] {
	case "list":
		vms := make([]tartVM, 0, len(f.vms))
		for _, vm := range f.vms {
			vms = append(vms, vm)
		}
		return json.Marshal(vms)
	case "clone":
		if _, ok := f.vms[args[1]]; !ok {
			return nil, fmt.Errorf("source %s does not exist", args[1])
		}
		f.vms[args[2]] = tartVM{Name: args[2], State: "stopped", Running: false}
	case "stop":
		if f.isStopStuck {
			return nil, errors.New("tart stop timed out")
		}
		f.vms[args[1]] = tartVM{Name: args[1], State: "stopped", Running: false}
		for pid, line := range f.processes {
			if strings.HasSuffix(line, " "+args[1]) {
				delete(f.processes, pid)
			}
		}
	case "delete":
		if f.vms[args[1]].Running {
			return nil, fmt.Errorf("%s is running", args[1])
		}
		delete(f.vms, args[1])
	}
	return nil, nil
}

func (f *fakeTart) startDetached(_ context.Context, name string, args []string, _ string) (int, error) {
	f.detached = append(f.detached, args)
	if f.failOn == "run" {
		return 0, errors.New("tart run failed")
	}
	vm := args[len(args)-1]
	f.vms[vm] = tartVM{Name: vm, State: "running", Running: true}
	f.nextPID++
	f.processes[f.nextPID] = name + " " + strings.Join(args, " ")
	return f.nextPID, nil
}

type harness struct {
	t        *testing.T
	tart     *fakeTart
	stateDir string
	config   string
}

func newHarness(t *testing.T, tart *fakeTart) harness {
	t.Helper()
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	cfgPath := filepath.Join(dir, "provider.toml")
	cfg := fmt.Sprintf("tart = \"/usr/bin/tart\"\nstate_dir = %q\ncpu = 4\nmemory = 8192\n\n[images]\nmacos-26 = \"golden-macos\"\n", stateDir)
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	return harness{t: t, tart: tart, stateDir: stateDir, config: cfgPath}
}

func (h harness) run(command, instanceID, pool string, stdin []byte) (int, string) {
	h.t.Helper()
	env := map[string]string{
		"GARM_COMMAND":              command,
		"GARM_PROVIDER_CONFIG_FILE": h.config,
		"GARM_CONTROLLER_ID":        testController,
		"GARM_INSTANCE_ID":          instanceID,
		"GARM_POOL_ID":              pool,
	}
	var stdout, stderr bytes.Buffer
	code := runProvider(h.t.Context(), func(key string) string { return env[key] }, bytes.NewReader(stdin), &stdout, &stderr, h.tart)
	return code, stdout.String()
}

func (h harness) create(name, image string) (int, string) {
	h.t.Helper()
	params := map[string]any{
		"name":           name,
		"image":          image,
		"pool_id":        testPool,
		"os_type":        "linux",
		"arch":           "arm64",
		"metadata-url":   "http://192.168.64.1:9997/api/v1/metadata/",
		"instance-token": "token-o'brien",
	}
	raw, err := json.Marshal(params)
	if err != nil {
		h.t.Fatal(err)
	}
	return h.run("CreateInstance", "", testPool, raw)
}

func (h harness) hasStateDir(name string) bool {
	_, err := os.Stat(filepath.Join(h.stateDir, "instances", name))
	return err == nil
}

func goldenVM() tartVM {
	return tartVM{Name: "golden-macos", State: "stopped", Running: false}
}

func TestCreateInstanceClonesGoldenAndStartsVMWithSharedDir(t *testing.T) {
	tart := newFakeTart(goldenVM())
	h := newHarness(t, tart)

	code, out := h.create("garm-one", "macos-26")

	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	var instance providerInstance
	if err := json.Unmarshal([]byte(out), &instance); err != nil {
		t.Fatal(err)
	}
	if instance.ProviderID != "garm-one" || instance.Name != "garm-one" || instance.Status != statusRunning || instance.OSArch != "arm64" {
		t.Fatalf("instance %+v", instance)
	}
	if !slices.Contains(tart.calls, "clone golden-macos garm-one") {
		t.Fatalf("calls %v", tart.calls)
	}
	if !slices.Contains(tart.calls, "set garm-one --random-mac --cpu 4 --memory 8192") {
		t.Fatalf("calls %v", tart.calls)
	}
	shared := filepath.Join(h.stateDir, "instances", "garm-one", "shared")
	want := []string{"run", "--no-graphics", "--no-audio", "--dir=garm:" + shared, "garm-one"}
	if len(tart.detached) != 1 || !slices.Equal(tart.detached[0], want) {
		t.Fatalf("detached %v", tart.detached)
	}
	script, err := os.ReadFile(filepath.Join(shared, "bootstrap.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(script), `metadata_url='http://192.168.64.1:9997/api/v1/metadata'`) ||
		!strings.Contains(string(script), `instance_token='token-o'\''brien'`) ||
		!strings.Contains(string(script), `"${metadata_url}/install-script/"`) {
		t.Fatalf("script:\n%s", script)
	}
	if _, err := os.Stat(filepath.Join(shared, "bootstrap.json")); err != nil {
		t.Fatal(err)
	}
	pid, err := os.ReadFile(filepath.Join(h.stateDir, "instances", "garm-one", "tart-run.pid"))
	if err != nil || strings.TrimSpace(string(pid)) != "101" {
		t.Fatalf("pid %q %v", pid, err)
	}
}

func TestCreateInstanceRejectsUnknownImage(t *testing.T) {
	tart := newFakeTart(goldenVM())
	h := newHarness(t, tart)

	code, _ := h.create("garm-one", "ubuntu")

	if code != 1 || h.hasStateDir("garm-one") || len(tart.calls) != 0 {
		t.Fatalf("code %d calls %v", code, tart.calls)
	}
}

func TestCreateInstanceRejectsPathInName(t *testing.T) {
	h := newHarness(t, newFakeTart(goldenVM()))

	code, _ := h.create("../escape", "macos-26")

	if code != 1 {
		t.Fatalf("exit code %d", code)
	}
}

func TestCreateInstanceRefusesExistingTartVM(t *testing.T) {
	tart := newFakeTart(goldenVM(), tartVM{Name: "garm-one", State: "stopped", Running: false})
	h := newHarness(t, tart)

	code, _ := h.create("garm-one", "macos-26")

	if code != exitCodeDuplicate || h.hasStateDir("garm-one") {
		t.Fatalf("exit code %d", code)
	}
	if _, ok := tart.vms["garm-one"]; !ok {
		t.Fatal("foreign VM was deleted")
	}
}

func TestCreateInstanceRemovesCloneAndStateWhenRunFails(t *testing.T) {
	tart := newFakeTart(goldenVM())
	tart.failOn = "run"
	h := newHarness(t, tart)

	code, _ := h.create("garm-one", "macos-26")

	if code != 1 {
		t.Fatalf("exit code %d", code)
	}
	if _, ok := tart.vms["garm-one"]; ok {
		t.Fatal("clone left behind")
	}
	if h.hasStateDir("garm-one") {
		t.Fatal("state left behind")
	}
}

func TestDeleteInstanceStopsAndDeletesCloneAndState(t *testing.T) {
	tart := newFakeTart(goldenVM())
	h := newHarness(t, tart)
	h.create("garm-one", "macos-26")

	code, _ := h.run("DeleteInstance", "garm-one", "", nil)

	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if _, ok := tart.vms["garm-one"]; ok {
		t.Fatal("clone left behind")
	}
	if h.hasStateDir("garm-one") {
		t.Fatal("state left behind")
	}
	if !slices.Contains(tart.calls, "stop garm-one --timeout 60") {
		t.Fatalf("calls %v", tart.calls)
	}
}

func TestDeleteInstanceLeavesVMWithoutStateAlone(t *testing.T) {
	tart := newFakeTart(goldenVM(), tartVM{Name: "other-vm", State: "running", Running: true})
	h := newHarness(t, tart)

	code, _ := h.run("DeleteInstance", "other-vm", "", nil)

	if code != 0 || len(tart.calls) != 0 {
		t.Fatalf("code %d calls %v", code, tart.calls)
	}
}

func TestDeleteInstanceRemovesStateWhenVMIsGone(t *testing.T) {
	tart := newFakeTart(goldenVM())
	h := newHarness(t, tart)
	h.create("garm-one", "macos-26")
	delete(tart.vms, "garm-one")

	code, _ := h.run("DeleteInstance", "garm-one", "", nil)

	if code != 0 || h.hasStateDir("garm-one") {
		t.Fatalf("exit code %d", code)
	}
}

func TestGetInstanceReportsTartState(t *testing.T) {
	tart := newFakeTart(goldenVM())
	h := newHarness(t, tart)
	h.create("garm-one", "macos-26")
	tart.vms["garm-one"] = tartVM{Name: "garm-one", State: "stopped", Running: false}

	code, out := h.run("GetInstance", "garm-one", "", nil)

	var instance providerInstance
	if err := json.Unmarshal([]byte(out), &instance); err != nil || code != 0 {
		t.Fatalf("code %d out %q", code, out)
	}
	if instance.Status != statusStopped {
		t.Fatalf("status %s", instance.Status)
	}
}

func TestGetInstanceReportsErrorWhenVMIsGone(t *testing.T) {
	tart := newFakeTart(goldenVM())
	h := newHarness(t, tart)
	h.create("garm-one", "macos-26")
	delete(tart.vms, "garm-one")

	_, out := h.run("GetInstance", "garm-one", "", nil)

	var instance providerInstance
	if err := json.Unmarshal([]byte(out), &instance); err != nil {
		t.Fatal(err)
	}
	if instance.Status != statusError || len(instance.ProviderFault) == 0 {
		t.Fatalf("instance %+v", instance)
	}
}

func TestGetInstanceOfMissingInstanceExitsNotFound(t *testing.T) {
	h := newHarness(t, newFakeTart(goldenVM()))

	code, _ := h.run("GetInstance", "garm-none", "", nil)

	if code != exitCodeNotFound {
		t.Fatalf("exit code %d", code)
	}
}

func TestListInstancesReturnsOnlyInstancesOfPoolAndController(t *testing.T) {
	tart := newFakeTart(goldenVM())
	h := newHarness(t, tart)
	h.create("garm-one", "macos-26")
	h.create("garm-two", "macos-26")
	store := stateStore{dir: h.stateDir}
	other, err := store.readInstance("garm-two")
	if err != nil {
		t.Fatal(err)
	}
	other.PoolID = "pool-b"
	if err := store.writeInstance(other); err != nil {
		t.Fatal(err)
	}

	code, out := h.run("ListInstances", "", testPool, nil)

	var instances []providerInstance
	if err := json.Unmarshal([]byte(out), &instances); err != nil || code != 0 {
		t.Fatalf("code %d out %q", code, out)
	}
	if len(instances) != 1 || instances[0].Name != "garm-one" {
		t.Fatalf("instances %+v", instances)
	}
}

func TestListInstancesPrintsEmptyArrayWithoutInstances(t *testing.T) {
	h := newHarness(t, newFakeTart(goldenVM()))

	_, out := h.run("ListInstances", "", testPool, nil)

	if strings.TrimSpace(out) != "[]" {
		t.Fatalf("out %q", out)
	}
}

func TestRemoveAllInstancesKeepsInstancesOfOtherControllers(t *testing.T) {
	tart := newFakeTart(goldenVM())
	h := newHarness(t, tart)
	h.create("garm-one", "macos-26")
	h.create("garm-two", "macos-26")
	store := stateStore{dir: h.stateDir}
	other, err := store.readInstance("garm-two")
	if err != nil {
		t.Fatal(err)
	}
	other.ControllerID = "controller-b"
	if err := store.writeInstance(other); err != nil {
		t.Fatal(err)
	}

	code, _ := h.run("RemoveAllInstances", "", "", nil)

	if code != 0 || h.hasStateDir("garm-one") || !h.hasStateDir("garm-two") {
		t.Fatalf("code %d", code)
	}
	if _, ok := tart.vms["garm-two"]; !ok {
		t.Fatal("instance of another controller was deleted")
	}
}

func TestStopAndStartInstance(t *testing.T) {
	tart := newFakeTart(goldenVM())
	h := newHarness(t, tart)
	h.create("garm-one", "macos-26")

	if code, _ := h.run("StopInstance", "garm-one", "", nil); code != 0 || tart.vms["garm-one"].Running {
		t.Fatalf("stop: code %d", code)
	}
	if code, _ := h.run("StartInstance", "garm-one", "", nil); code != 0 || !tart.vms["garm-one"].Running {
		t.Fatalf("start: code %d", code)
	}
	if len(tart.detached) != 2 {
		t.Fatalf("detached %v", tart.detached)
	}
}

func TestUnknownCommandFails(t *testing.T) {
	h := newHarness(t, newFakeTart())

	code, _ := h.run("Reboot", "", "", nil)

	if code != 1 {
		t.Fatalf("exit code %d", code)
	}
}

func TestGetVersionPrintsInterfaceVersion(t *testing.T) {
	h := newHarness(t, newFakeTart())

	_, out := h.run("GetVersion", "", "", nil)

	if strings.TrimSpace(out) != interfaceVersion {
		t.Fatalf("out %q", out)
	}
}

func TestStopInstanceKillsTartRunThatOutlivesTartStop(t *testing.T) {
	tart := newFakeTart(goldenVM())
	h := newHarness(t, tart)
	h.create("garm-one", "macos-26")
	tart.isStopStuck = true

	err := h.createProvider().stopInstance(t.Context(), "garm-one")

	if err != nil || !slices.Equal(tart.killed, []int{101}) {
		t.Fatalf("err %v killed %v", err, tart.killed)
	}
}

func TestStopInstanceLeavesProcessOfAnotherVMAlone(t *testing.T) {
	tart := newFakeTart(goldenVM())
	h := newHarness(t, tart)
	h.create("garm-one", "macos-26")
	tart.isStopStuck = true
	tart.processes[101] = "/usr/bin/tart run --no-graphics garm-one0"

	err := h.createProvider().stopInstance(t.Context(), "garm-one")

	if err != nil || len(tart.killed) != 0 {
		t.Fatalf("err %v killed %v", err, tart.killed)
	}
}

func (h harness) createProvider() provider {
	h.t.Helper()
	cfg, err := readConfig(h.config)
	if err != nil {
		h.t.Fatal(err)
	}
	return provider{
		cfg:             cfg,
		tart:            tartClient{runner: h.tart, binary: cfg.Tart},
		processes:       processTable{runner: h.tart},
		store:           stateStore{dir: h.stateDir},
		controllerID:    testController,
		exitWaitSeconds: 0,
	}
}
