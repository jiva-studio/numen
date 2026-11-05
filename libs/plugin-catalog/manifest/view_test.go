package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
)

func TestAManifestDeclaresTheTitleBarViewOfThePlugin(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(contributesHead + `,"contributes":{
		"views":[{"id":"bar","slot":"titleBar"}]}}`))
	if err != nil {
		t.Fatal(err)
	}

	view, ok := m.Contributes.GetSlotView(manifest.SlotTitleBar)
	if !ok || view.ID != "bar" {
		t.Errorf("the title bar view = %+v, %v", view, ok)
	}
	if _, ok := m.Contributes.GetSlotView("statusBar"); ok {
		t.Error("a view is found for a slot the manifest does not declare")
	}
}

func TestAViewTheApplicationCannotUseIsRejected(t *testing.T) {
	cases := map[string]string{
		"an id in capitals":          `{"views":[{"id":"Bar","slot":"titleBar"}]}`,
		"an id with no name":         `{"views":[{"id":"","slot":"titleBar"}]}`,
		"an id declared twice":       `{"views":[{"id":"bar","slot":"titleBar"},{"id":"bar","slot":"titleBar"}]}`,
		"a slot the window lacks":    `{"views":[{"id":"bar","slot":"sideBar"}]}`,
		"a view with no slot":        `{"views":[{"id":"bar"}]}`,
		"a slot declared by two ids": `{"views":[{"id":"bar","slot":"titleBar"},{"id":"strip","slot":"titleBar"}]}`,
	}
	for name, contributes := range cases {
		_, err := manifestjson.Read(strings.NewReader(contributesHead + `,"contributes":` + contributes + `}`))

		if !errors.Is(err, manifest.ErrInvalidContribution) {
			t.Errorf("%s: error = %v, want an invalid contribution", name, err)
		}
	}
}

func TestAViewNeedsAFrontendToRegisterIt(t *testing.T) {
	_, err := manifestjson.Read(strings.NewReader(
		`{"id":"a.b","name":"A","version":"1.0.0","contributes":{"views":[{"id":"bar","slot":"titleBar"}]}}`))

	if !errors.Is(err, manifest.ErrInvalidContribution) || !strings.Contains(err.Error(), "no frontend") {
		t.Errorf("error = %v, want an error for the missing frontend", err)
	}
}
