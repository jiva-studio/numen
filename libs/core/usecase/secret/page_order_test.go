package secret

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

type pageOrderCase struct {
	Name    string   `json:"name"`
	Asked   []string `json:"asked"`
	Secrets []struct {
		ID    string `json:"id"`
		Group string `json:"group"`
	} `json:"secrets"`
	Expected []string `json:"expected"`
}

// The page of secrets in the window orders the sections the same way, from the same cases.
func TestTheOrderOfThePageIsTheOneTheWindowDraws(t *testing.T) {
	raw, err := os.ReadFile("testdata/page_order.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []pageOrderCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, one := range cases {
		t.Run(one.Name, func(t *testing.T) {
			var list []Secret
			for _, entry := range one.Secrets {
				list = append(list, Secret{ID: entry.ID, Name: entry.ID, Group: entry.Group})
			}
			declarations := make([]Permission, 0, len(one.Asked))
			for _, group := range one.Asked {
				declarations = append(declarations, Permission{Plugin: "p", Name: "apiKey", Group: group})
			}
			var got []string
			for _, entry := range ListInPageOrder(list, ListAskedGroups(declarations)) {
				got = append(got, entry.ID)
			}
			if !slices.Equal(got, one.Expected) {
				t.Errorf("order = %v, want %v", got, one.Expected)
			}
		})
	}
}
