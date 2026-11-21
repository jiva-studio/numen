package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func readSlotPoints(points, settings string) (manifest.Manifest, error) {
	return manifestjson.Read(strings.NewReader(`{"id":"org.example.ledger","name":"Ledger","version":"1.0.0","frontend":{},
		"settings":[` + settings + `],"contributes":{"slotPoints":` + points + `}}`))
}

const slotsSetting = `{"key":"slots","type":"slots","title":"Slots"}`

func TestSlotPointsRead(t *testing.T) {
	parsed, err := readSlotPoints(`[{"slot":"ledger.rowEnd","title":"End of a row"},{"slot":"ledger.context","title":"Menu","menu":true}]`, slotsSetting)
	if err != nil {
		t.Fatal(err)
	}
	got := parsed.Contributes.SlotPoints
	if len(got) != 2 || got[0].Slot != "ledger.rowEnd" || got[0].Title != "End of a row" || got[0].Menu || !got[1].Menu {
		t.Errorf("slot points came out as %+v", got)
	}
}

func TestSlotPointsChecked(t *testing.T) {
	for name, parts := range map[string][2]string{
		"bad name":      {`[{"slot":"Ledger","title":"A"}]`, slotsSetting},
		"twice":         {`[{"slot":"a.b","title":"A"},{"slot":"a.b","title":"B"}]`, slotsSetting},
		"no title":      {`[{"slot":"a.b","title":" "}]`, slotsSetting},
		"no setting":    {`[{"slot":"a.b","title":"A"}]`, ``},
		"no slot point": {`[]`, slotsSetting},
	} {
		if _, err := readSlotPoints(parts[0], parts[1]); !errors.Is(err, manifest.ErrInvalidSlotPoint) {
			t.Errorf("%s gave %v", name, err)
		}
	}
}
