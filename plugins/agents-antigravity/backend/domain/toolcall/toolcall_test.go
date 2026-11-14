package toolcall

import "testing"

func TestTheSubjectOfACallIsItsQueryElseItsPath(t *testing.T) {
	for _, tc := range []struct{ arguments, about, path string }{
		{`{"path":"Garden/Beds.md","start":4}`, "Garden/Beds.md", "Garden/Beds.md"},
		{`{"file":"Orchard.md","query":"pear trees"}`, "pear trees", "Orchard.md"},
		{`{"title":"Compost","about":"layers"}`, "layers", ""},
		{`{"query":"","title":"Seed list"}`, "Seed list", ""},
		{`{"count":3}`, "", ""},
		{`["Garden.md"]`, "", ""},
		{`{"path":`, "", ""},
		{``, "", ""},
	} {
		about, path := ReadSubject([]byte(tc.arguments))
		if about != tc.about || path != tc.path {
			t.Errorf("ReadSubject(%s) = %q, %q, want %q, %q", tc.arguments, about, path, tc.about, tc.path)
		}
	}
}
