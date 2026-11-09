package settings_test

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
)

const (
	tasksKey   = "org.example.tide/status"
	statsKey   = "org.example.reef/stats"
	settingKey = "app/app:settings.open"
)

func slotPath(slot string) []string { return []string{settings.OwnerApp, "layout", "slots", slot} }

func isRefusedWrite(err error) bool {
	return errors.Is(err, settings.ErrUndeclared) || errors.Is(err, settings.ErrWrongType)
}

func TestAListOfEntryKeysIsAcceptedForEachBarSlot(t *testing.T) {
	for _, slot := range []string{"titleBar.commands", "titleBar.status", "statusBar.left", "statusBar.right"} {
		if err := check(slotPath(slot), []any{tasksKey, statsKey, settingKey}); err != nil {
			t.Errorf("a list for %s was refused: %v", slot, err)
		}
	}
}

func TestAKeyOfAPluginThatIsNotInstalledIsAcceptedInALayout(t *testing.T) {
	if err := check(slotPath("statusBar.left"), []any{"org.example.nowhere/clock"}); err != nil {
		t.Errorf("the key of a missing plugin was refused: %v", err)
	}
}

func TestTheWholeLayoutIsAcceptedAsTheMapOfSlotsOrAsTheLayoutObject(t *testing.T) {
	slots := map[string]any{"titleBar.status": []any{tasksKey}, "statusBar.right": []any{statsKey}}
	if err := check([]string{settings.OwnerApp, "layout", "slots"}, slots); err != nil {
		t.Errorf("the map of slots was refused: %v", err)
	}
	if err := check([]string{settings.OwnerApp, "layout"}, map[string]any{"slots": slots}); err != nil {
		t.Errorf("the layout object was refused: %v", err)
	}
}

func TestAWrongLayoutWriteIsRefusedWhereTheRightOneIsAccepted(t *testing.T) {
	if err := check(slotPath("titleBar.status"), []any{tasksKey, statsKey}); err != nil {
		t.Fatalf("the right write was refused: %v", err)
	}
	for what, c := range map[string]struct {
		path  []string
		value any
	}{
		"a list under a menu slot":        {slotPath("fileTree.context"), []any{tasksKey}},
		"a list under an unknown slot":    {slotPath("sideBar.top"), []any{tasksKey}},
		"a string instead of a list":      {slotPath("titleBar.status"), tasksKey},
		"a list holding a number":         {slotPath("titleBar.status"), []any{tasksKey, 3.0}},
		"a key with no slash":             {slotPath("titleBar.status"), []any{"org.example.tide"}},
		"a key with an empty id":          {slotPath("titleBar.status"), []any{"org.example.tide/"}},
		"a key whose owner is no plugin":  {slotPath("titleBar.status"), []any{"Tide Plugin/status"}},
		"a key twice in one list":         {slotPath("titleBar.status"), []any{tasksKey, statsKey, tasksKey}},
		"a map with a menu slot":          {[]string{settings.OwnerApp, "layout", "slots"}, map[string]any{"fileTree.context": []any{tasksKey}}},
		"a map holding a string":          {[]string{settings.OwnerApp, "layout", "slots"}, map[string]any{"titleBar.status": tasksKey}},
		"a layout with another member":    {[]string{settings.OwnerApp, "layout"}, map[string]any{"slots": map[string]any{}, "hidden": []any{}}},
		"a path below the list of a slot": {append(slotPath("titleBar.status"), "0"), tasksKey},
		"a path beside the slots":         {[]string{settings.OwnerApp, "layout", "hidden"}, []any{tasksKey}},
		"slots that are not an object":    {[]string{settings.OwnerApp, "layout", "slots"}, []any{tasksKey}},
	} {
		if err := check(c.path, c.value); !isRefusedWrite(err) {
			t.Errorf("%s gave %v, want a refusal", what, err)
		}
	}
}

func readLayoutInForce(t *testing.T, cfg interface {
	GetValue(path ...string) (any, bool)
}) string {
	t.Helper()
	value, found := cfg.GetValue(settings.OwnerApp, "layout", "slots")
	if !found {
		return ""
	}
	content, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func TestAValidLayoutInTheFileIsInForceWithoutANotice(t *testing.T) {
	cfg, err := openAt(write(t, `{"app":{"layout":{"slots":{"titleBar.status":["`+tasksKey+`","`+statsKey+`"]}}}}`))
	if err != nil {
		t.Fatalf("the file was refused: %v", err)
	}

	if got, want := readLayoutInForce(t, &cfg), `{"titleBar.status":["`+tasksKey+`","`+statsKey+`"]}`; got != want {
		t.Errorf("the layout in force is %s, want %s", got, want)
	}
	if len(cfg.Notices) != 0 {
		t.Errorf("the notices are %q", cfg.Notices)
	}
}

func TestALayoutWithAMenuSlotIsReadWithANoticeAndItsBarSlotsStayInForce(t *testing.T) {
	const body = `{"app":{"layout":{"slots":{"fileTree.context":["` + tasksKey + `"],"titleBar.status":["` + statsKey + `"]}}}}`
	path := write(t, body)

	cfg, err := openAt(path)
	if err != nil {
		t.Fatalf("the file was refused: %v", err)
	}

	if !hasNotice(cfg.Notices, "app.layout.slots.fileTree.context") {
		t.Errorf("the notices are %q", cfg.Notices)
	}
	if hasNotice(cfg.Notices, "titleBar.status") {
		t.Errorf("the valid list has a notice: %q", cfg.Notices)
	}
	if got, want := readLayoutInForce(t, &cfg), `{"titleBar.status":["`+statsKey+`"]}`; got != want {
		t.Errorf("the layout in force is %s, want %s", got, want)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != body {
		t.Errorf("the file is now:\n%s", raw)
	}
}

func TestALayoutListThatIsNotAListOfStringsIsIgnoredWithANotice(t *testing.T) {
	cfg, err := openAt(write(t, `{"app":{"layout":{"slots":{"statusBar.left":"`+tasksKey+`","statusBar.right":["`+statsKey+`",3],"titleBar.status":["`+tasksKey+`"]}}}}`))
	if err != nil {
		t.Fatalf("the file was refused: %v", err)
	}

	for _, slot := range []string{"app.layout.slots.statusBar.left", "app.layout.slots.statusBar.right"} {
		if !hasNotice(cfg.Notices, slot) {
			t.Errorf("no notice names %s: %q", slot, cfg.Notices)
		}
	}
	if got, want := readLayoutInForce(t, &cfg), `{"titleBar.status":["`+tasksKey+`"]}`; got != want {
		t.Errorf("the layout in force is %s, want %s", got, want)
	}
}

func TestALayoutKeyWithNoOwnerIsLeftOutWithANotice(t *testing.T) {
	cfg, err := openAt(write(t, `{"app":{"layout":{"slots":{"titleBar.status":["`+tasksKey+`","status"]}}}}`))
	if err != nil {
		t.Fatalf("the file was refused: %v", err)
	}

	if !hasNotice(cfg.Notices, "app.layout.slots.titleBar.status") {
		t.Errorf("the notices are %q", cfg.Notices)
	}
	value, _ := cfg.GetValue(settings.OwnerApp, "layout", "slots", "titleBar.status")
	if list, isList := value.([]any); isList {
		for _, key := range list {
			if key == "status" {
				t.Errorf("the key without an owner is in force: %v", list)
			}
		}
	}
}
