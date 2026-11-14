package manifest_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestAListSettingWithADefaultOfDistinctStringsIsAccepted(t *testing.T) {
	m, err := readSettings(`{"key":"folders","type":"list","title":"Folders","default":["a","b"]}`)
	if err != nil {
		t.Fatalf("err = %v, want none", err)
	}

	declared, ok := m.GetSetting("folders")
	if !ok || declared.Type != manifest.SettingTypeList {
		t.Fatalf("the list setting = %+v, %v", declared, ok)
	}
	if !reflect.DeepEqual(declared.Default, []any{"a", "b"}) {
		t.Errorf("Default = %#v, want [a b]", declared.Default)
	}
}

func TestAListSettingWithoutADefaultIsAccepted(t *testing.T) {
	_, err := readSettings(`{"key":"folders","type":"list","title":"Folders"}`)

	if err != nil {
		t.Errorf("err = %v, want none", err)
	}
}

func TestAListSettingWhoseDefaultIsAStringIsRefused(t *testing.T) {
	_, err := readSettings(`{"key":"folders","type":"list","title":"Folders","default":"a,b"}`)

	if !errors.Is(err, manifest.ErrInvalidSetting) || !strings.Contains(err.Error(), "folders holds a list and its default is no list of strings") {
		t.Errorf("err = %v, want an issue that the default is no list of strings", err)
	}
}

func TestAListSettingWhoseDefaultHoldsANumberIsRefused(t *testing.T) {
	_, err := readSettings(`{"key":"folders","type":"list","title":"Folders","default":["a",1]}`)

	if !errors.Is(err, manifest.ErrInvalidSetting) || !strings.Contains(err.Error(), "folders holds a list and its default is no list of strings") {
		t.Errorf("err = %v, want an issue that the default is no list of strings", err)
	}
}

func TestAListSettingWhoseDefaultHoldsAnEmptyStringIsRefused(t *testing.T) {
	for _, item := range []string{``, ` `} {
		_, err := readSettings(`{"key":"folders","type":"list","title":"Folders","default":["a","` + item + `"]}`)

		if !errors.Is(err, manifest.ErrInvalidSetting) || !strings.Contains(err.Error(), "folders holds a list and its default holds an empty item") {
			t.Errorf("%q: err = %v, want an issue that the default holds an empty item", item, err)
		}
	}
}

func TestAListSettingWhoseDefaultHoldsOneValueTwiceIsRefused(t *testing.T) {
	_, err := readSettings(`{"key":"folders","type":"list","title":"Folders","default":["a","b","a"]}`)

	if !errors.Is(err, manifest.ErrInvalidSetting) || !strings.Contains(err.Error(), `folders holds a list and its default holds "a" twice`) {
		t.Errorf("err = %v, want an issue that the default holds a twice", err)
	}
}

func TestAListSettingWhoseDefaultHoldsTwoValuesThatDifferInCaseIsAccepted(t *testing.T) {
	_, err := readSettings(`{"key":"folders","type":"list","title":"Folders","default":["Archive","archive"]}`)

	if err != nil {
		t.Errorf("err = %v, want none", err)
	}
}

func TestAListSettingThatDeclaresAFieldOfAnotherTypeIsRefused(t *testing.T) {
	for field, extra := range map[string]string{
		"options":     `"options":["x"]`,
		"optionsBy":   `"optionsBy":{"setting":"mode","options":{"a":["x"]}}`,
		"optionsFrom": `"optionsFrom":"themes"`,
		"min":         `"min":1`,
		"max":         `"max":2`,
		"step":        `"step":1`,
		"accept":      `"accept":[".txt"]`,
		"multiple":    `"multiple":true`,
		"preview":     `"preview":true`,
		"palette":     `"palette":true`,
	} {
		_, err := readSettings(`{"key":"mode","type":"enum","title":"Mode","options":["a"]},` +
			`{"key":"folders","type":"list","title":"Folders",` + extra + `}`)

		if !errors.Is(err, manifest.ErrInvalidSetting) || !strings.Contains(err.Error(), "folders holds a list and declares "+field) {
			t.Errorf("%s: err = %v, want an issue that the list declares %s", field, err, field)
		}
	}
}

func TestAListSettingThatDeclaresTwoFieldsOfAnotherTypeIsRefusedForEach(t *testing.T) {
	_, err := readSettings(`{"key":"folders","type":"list","title":"Folders","min":1,"step":1}`)

	if err == nil || !strings.Contains(err.Error(), "folders holds a list and declares min") || !strings.Contains(err.Error(), "folders holds a list and declares step") {
		t.Errorf("err = %v, want one issue for min and one for step", err)
	}
}

func TestASettingShownByAListSettingIsRefused(t *testing.T) {
	_, err := readSettings(`{"key":"folders","type":"list","title":"Folders"},` +
		`{"key":"depth","type":"number","title":"Depth","when":{"setting":"folders","is":["a"]}}`)

	if !errors.Is(err, manifest.ErrInvalidSetting) || !strings.Contains(err.Error(), "depth is shown by folders, which holds a list") {
		t.Errorf("err = %v, want an issue that depth is shown by a list", err)
	}
}
