package machine

import "testing"

func TestTheDocumentsFolderIsReadFromTheUserDirsFile(t *testing.T) {
	cases := map[string]struct {
		content string
		want    string
		isFound bool
	}{
		"under home":     {"XDG_DESKTOP_DIR=\"$HOME/Desktop\"\nXDG_DOCUMENTS_DIR=\"$HOME/Papers\"\n", "/home/ann/Papers", true},
		"absolute":       {"XDG_DOCUMENTS_DIR=\"/data/docs\"\n", "/data/docs", true},
		"not named":      {"XDG_DESKTOP_DIR=\"$HOME/Desktop\"\n", "", false},
		"relative":       {"XDG_DOCUMENTS_DIR=\"docs\"\n", "", false},
		"commented out":  {"# XDG_DOCUMENTS_DIR=\"$HOME/Papers\"\n", "", false},
		"spaced equals":  {"XDG_DOCUMENTS_DIR = \"$HOME/Papers\"\n", "/home/ann/Papers", true},
		"trailing slash": {"XDG_DOCUMENTS_DIR=\"$HOME/Papers/\"\n", "/home/ann/Papers", true},
	}
	for name, one := range cases {
		got, isFound := parseDocumentsFolder(one.content, "/home/ann")
		if got != one.want || isFound != one.isFound {
			t.Errorf("%s: got %q, %v, want %q, %v", name, got, isFound, one.want, one.isFound)
		}
	}
}
