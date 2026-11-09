package container_test

import (
	"encoding/json"
	"errors"
	"maps"
	"os"
	"reflect"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/port/settingsport"
)

const (
	layoutTasks   = "titleBar.status|org.example.tide/status"
	layoutStats   = "titleBar.status|org.example.reef/stats"
	layoutUpdater = "titleBar.status|org.example.harbor/update"
)

func layoutSlotsPath() []string { return []string{"app", "layout", "slots"} }

func writeLayoutSlots(cfg container.Config, lists map[string]string) error {
	var written []settingsport.Setting
	for _, slot := range slices.Sorted(maps.Keys(lists)) {
		written = append(written, settingsport.Setting{Path: append(layoutSlotsPath(), slot), JSON: lists[slot]})
	}
	return cfg.GetSettingsWriter()(written)
}

func decodeLayoutSlots(t *testing.T, content string) map[string][]string {
	t.Helper()
	var file struct {
		App struct {
			Layout struct {
				Slots map[string][]string `json:"slots"`
			} `json:"layout"`
		} `json:"app"`
	}
	if err := json.Unmarshal([]byte(content), &file); err != nil {
		t.Fatalf("%v in:\n%s", err, content)
	}
	return file.App.Layout.Slots
}

func decodeSlotsChange(t *testing.T, batch []settingsport.Setting) map[string][]string {
	t.Helper()
	if len(batch) != 1 || !slices.Equal(batch[0].Path, layoutSlotsPath()) {
		t.Fatalf("the change is %v, want one change of %v", batch, layoutSlotsPath())
	}
	var slots map[string][]string
	if err := json.Unmarshal([]byte(batch[0].JSON), &slots); err != nil {
		t.Fatalf("%v in %s", err, batch[0].JSON)
	}
	return slots
}

func TestAMoveWritesBothSlotListsIntoTheFileAndReportsTheWholeLayoutOnce(t *testing.T) {
	cfg, path, _ := openInstallation(t, `{}`)
	reader := listen(t, cfg)
	want := map[string][]string{
		"titleBar.status": {layoutTasks, layoutStats},
		"statusBar.right": {layoutUpdater},
	}

	err := writeLayoutSlots(cfg, map[string]string{
		"titleBar.status": `["` + layoutTasks + `","` + layoutStats + `"]`,
		"statusBar.right": `["` + layoutUpdater + `"]`,
	})
	if err != nil {
		t.Fatalf("the write gave %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := decodeLayoutSlots(t, string(raw)); !reflect.DeepEqual(got, want) {
		t.Errorf("the file holds %v, want %v", got, want)
	}
	content, _, err := cfg.ReadSettings()()
	if err != nil {
		t.Fatal(err)
	}
	if got := decodeLayoutSlots(t, content); !reflect.DeepEqual(got, want) {
		t.Errorf("the written settings hold %v, want %v", got, want)
	}
	if got := decodeSlotsChange(t, reader.await(t)); !reflect.DeepEqual(got, want) {
		t.Errorf("the reported layout is %v, want %v", got, want)
	}
	reader.expectNothing(t)
}

func TestALayoutEditedByHandReachesEveryReader(t *testing.T) {
	cfg, path, _ := openInstallation(t, `{"app":{"layout":{"slots":{"titleBar.status":["`+layoutStats+`"]}}}}`)
	reader := listen(t, cfg)

	body := `{"app":{"layout":{"slots":{"titleBar.status":["` + layoutTasks + `","` + layoutStats + `"],"statusBar.left":["` + layoutUpdater + `"]}}}}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	want := map[string][]string{
		"titleBar.status": {layoutTasks, layoutStats},
		"statusBar.left":  {layoutUpdater},
	}
	if got := decodeSlotsChange(t, reader.await(t)); !reflect.DeepEqual(got, want) {
		t.Errorf("the reported layout is %v, want %v", got, want)
	}
}

func TestAWrongLayoutWriteIsRefusedAndLeavesTheFileAsItWas(t *testing.T) {
	for what, lists := range map[string]map[string]string{
		"a list under an undeclared slot": {"sideBar.top": `["` + layoutTasks + `"]`},
		"a string instead of a list":      {"titleBar.status": `"` + layoutTasks + `"`},
		"a list holding a number":         {"titleBar.status": `["` + layoutTasks + `",3]`},
		"a key with no slash":             {"titleBar.status": `["titleBar.status|org.example.tide"]`},
		"a key twice in one list":         {"titleBar.status": `["` + layoutTasks + `","` + layoutTasks + `"]`},
		"a move with one wrong list":      {"titleBar.status": `["` + layoutStats + `"]`, "statusBar.right": `["` + layoutUpdater + `",1]`},
	} {
		cfg, path, _ := openInstallation(t, `{}`)

		if err := writeLayoutSlots(cfg, lists); !errors.Is(err, settingsport.ErrNotASetting) {
			t.Errorf("%s gave %v", what, err)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(raw) != `{}` {
			t.Errorf("%s left the file as:\n%s", what, raw)
		}
		if err := writeLayoutSlots(cfg, map[string]string{"titleBar.status": `["` + layoutStats + `"]`}); err != nil {
			t.Errorf("after %s, the right list was refused: %v", what, err)
		}
	}
}
