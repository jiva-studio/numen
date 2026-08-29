package settings

import "testing"

// A rename replaces one word of the file where it is written, so what it will
// not touch is worth as much as what it does. A settings file is written into
// by a person and by the window, and what one of them wrote between a read and
// a write is still there when the write lands.
func TestWhatIsNotRenamed(t *testing.T) {
	for what, held := range map[string]struct {
		file string
		at   []string
	}{
		"a section already holding the name": {
			`{"appearance":{"zoom":1.5,"interface_scale":1.25}}`, []string{"appearance", "zoom"},
		},
		"a name the file has not got": {
			`{"appearance":{"theme":"preset:numen"}}`, []string{"appearance", "zoom"},
		},
		"a section the file has not got": {
			`{"agent":{"use":"claude"}}`, []string{"appearance", "zoom"},
		},
		"a section held as something else": {
			`{"appearance":"warm"}`, []string{"appearance", "zoom"},
		},
		// A name is replaced with the bytes of the name it is read as. Where
		// those are not the bytes the file holds, the span to replace is not
		// known and the file is left alone.
		"a name written otherwise than it is read": {
			`{"appearance":{"zo<om":1.5}}`, []string{"appearance", "zo<om"},
		},
		"a name with no section named": {
			`{"appearance":{"zoom":1.5}}`, nil,
		},
	} {
		back, done := named([]byte(held.file), held.at, "interface_scale")
		if done {
			t.Errorf("%s: renamed, leaving %s", what, back)
		}
		if string(back) != held.file {
			t.Errorf("%s: the file is now %s", what, back)
		}
	}
}

// The name is replaced where it is written and the value is left where it sits,
// whatever lies between the two.
func TestARenamedFieldKeepsItsValueAndItsPlace(t *testing.T) {
	const held = "{\n  \"appearance\": {\n    \"zoom\"  :  1.50,\n    \"mode\": \"dark\"\n  }\n}\n"
	back, done := named([]byte(held), []string{"appearance", "zoom"}, "interface_scale")
	if !done {
		t.Fatal("nothing was renamed")
	}
	want := "{\n  \"appearance\": {\n    \"interface_scale\"  :  1.50,\n    \"mode\": \"dark\"\n  }\n}\n"
	if string(back) != want {
		t.Errorf("the file came back as:\n%s\nand not as:\n%s", back, want)
	}
}
