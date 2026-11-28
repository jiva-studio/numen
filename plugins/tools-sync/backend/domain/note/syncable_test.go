package note_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
)

func TestIsSyncable_AcceptsFilesOfAnyKindAndRefusesFoldersAndNamesThatAreNotThePersons(t *testing.T) {
	t.Parallel()

	for _, one := range []struct {
		path    string
		isValid bool
	}{
		{"Papers/Report.pdf", true},
		{"Pictures/Tree.png", true},
		{"Recordings/Walk.m4a", true},
		{"Notes/Milk.md", true},
		{"Archive/data.unknownext", true},
		{"no-extension", true},
		{"Papers/.hidden-but-the-persons", true},
		{".numen/tools-sync/state.json", false},
		{"Papers/.numen/x.pdf", false},
		{".git/config", false},
		{"Code/.git/HEAD", false},
		{".trash/Old.pdf", false},
		{"Papers/.trash/Old.pdf", false},
		{".GIT/hooks/x", false},
		{".Numen/config.json", false},
		{"notes/.Git/x", false},
		{".TRASH/a.md", false},
		{".DS_Store", false},
		{"Papers/.DS_Store", false},
		{"Pictures/Thumbs.db", false},
		{"Pictures/desktop.ini", false},
		{"Papers/Report.pdf.part", false},
		{"Papers/Report.pdf.tmp", false},
		{"Notes/Milk.md~", false},
		{"Papers/~$Report.docx", false},
		{"Papers/Report~1.docx", true},
		{"Papers/The .numen guide.pdf", true},
		{strings.Repeat("a", 1000), true},
		{strings.Repeat("a", 1001), false},
	} {
		if got := note.IsSyncable(one.path); got != one.isValid {
			t.Errorf("IsSyncable(%q) = %v, want %v", one.path, got, one.isValid)
		}
	}
}

func TestKind_NamesMarkdownTextAndBinaryByExtension(t *testing.T) {
	t.Parallel()

	for path, want := range map[string]string{
		"Notes/Milk.md":        note.KindMarkdown,
		"Notes/Milk.MD":        note.KindMarkdown,
		"Data/list.txt":        note.KindText,
		"Data/list.json":       note.KindText,
		"Data/list.csv":        note.KindText,
		"Data/list.xml":        note.KindText,
		"Web/page.html":        note.KindText,
		"Web/logo.svg":         note.KindText,
		"Config/app.yaml":      note.KindText,
		"Config/app.yml":       note.KindText,
		"Config/app.toml":      note.KindText,
		"Logs/run.log":         note.KindText,
		"Data/LIST.TXT":        note.KindText,
		"Papers/Report.pdf":    note.KindBinary,
		"Pictures/Tree.png":    note.KindBinary,
		"Recordings/Walk.m4a":  note.KindBinary,
		"Archive/a.tar.gz":     note.KindBinary,
		"no-extension":         note.KindBinary,
		"Papers/Report.pdf.md": note.KindMarkdown,
	} {
		if got := note.GetKind(path); got != want {
			t.Errorf("GetKind(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestLimits_HaveTheValuesOfTheSpecification(t *testing.T) {
	t.Parallel()

	for name, one := range map[string][2]int{
		"BigFileLimit":       {note.BigFileLimit, 64 << 20},
		"MassDeleteRatio":    {note.MassDeleteRatio, 30},
		"MassDeleteMinFiles": {note.MassDeleteMinFiles, 5},
		"BatchFiles":         {note.BatchFiles, 100},
		"MovedBatchFiles":    {note.MovedBatchFiles, 5000},
	} {
		if one[0] != one[1] {
			t.Errorf("%s = %d, want %d", name, one[0], one[1])
		}
	}
}
