package tags_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/domain/tags"
)

func TestApplyTogglesTags(t *testing.T) {
	for name, c := range map[string]struct {
		current []string
		toggles []string
		want    []string
	}{
		"a tag the note has not is appended":                {current: []string{"Port"}, toggles: []string{"Sea"}, want: []string{"Port", "Sea"}},
		"a tag the note has is removed":                     {current: []string{"Port", "Sea"}, toggles: []string{"Port"}, want: []string{"Sea"}},
		"the other tags keep their order":                   {current: []string{"Port", "Sea", "Quay"}, toggles: []string{"Sea"}, want: []string{"Port", "Quay"}},
		"new tags are appended in the order of the toggles": {current: []string{"Port"}, toggles: []string{"Sea", "Wharf"}, want: []string{"Port", "Sea", "Wharf"}},
		"a tag is set and another is cleared together":      {current: []string{"Port", "Sea"}, toggles: []string{"Port", "Wharf"}, want: []string{"Sea", "Wharf"}},
		"the last tag removed leaves none":                  {current: []string{"Port"}, toggles: []string{"Port"}, want: nil},
		"a note with no tags gets the toggles":              {current: nil, toggles: []string{"Port", "Sea"}, want: []string{"Port", "Sea"}},
		"no toggles leave the tags as they are":             {current: []string{"Port", "Sea"}, toggles: nil, want: []string{"Port", "Sea"}},
		"a toggle in another case removes the tag":          {current: []string{"Port", "Sea"}, toggles: []string{"port"}, want: []string{"Sea"}},
		"a toggle in another Unicode form removes the tag":  {current: []string{"Café"}, toggles: []string{"Café"}, want: nil},
		"a hash in front of a toggle is dropped":            {current: []string{"Port"}, toggles: []string{"#Wharf"}, want: []string{"Port", "Wharf"}},
		"a hash in front of a toggle still finds the tag":   {current: []string{"Port", "Sea"}, toggles: []string{"#Port"}, want: []string{"Sea"}},
		"a nested tag is compared whole":                    {current: []string{"Projects/Web"}, toggles: []string{"projects/web"}, want: nil},
		"a parent of a nested tag is another tag":           {current: []string{"Projects/Web"}, toggles: []string{"Projects"}, want: []string{"Projects/Web", "Projects"}},
		"a toggle that cleans to nothing is ignored":        {current: []string{"Port"}, toggles: []string{"#", "  ", "/"}, want: []string{"Port"}},
		"a tag keeps the spelling the note has":             {current: []string{"PORT", "Sea"}, toggles: []string{"Wharf"}, want: []string{"PORT", "Sea", "Wharf"}},
		"a new tag is spelled as it is cleaned":             {current: nil, toggles: []string{" #Wharf / Quay "}, want: []string{"Wharf/Quay"}},
	} {
		if got := tags.Apply(c.current, c.toggles); !slices.Equal(got, c.want) {
			t.Errorf("%s: Apply(%q, %q) = %q, want %q", name, c.current, c.toggles, got, c.want)
		}
	}
}

func TestApplyDoesNotModifyItsInput(t *testing.T) {
	current := []string{"Port", "Sea", "Quay"}

	got := tags.Apply(current, []string{"Sea", "Wharf"})

	if !slices.Equal(current, []string{"Port", "Sea", "Quay"}) {
		t.Errorf("the list given is %q, want it unchanged", current)
	}
	if !slices.Equal(got, []string{"Port", "Quay", "Wharf"}) {
		t.Errorf("Apply returned %q, want the tags after the toggles", got)
	}
}
