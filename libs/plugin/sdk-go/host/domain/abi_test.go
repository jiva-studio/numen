package domain_test

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// The plugin SDK hands plugins the value types of the notes library as they are: a plugin is built
// against the SDK and the library of one release, and the types cross no process boundary as Go values.
// The shapes below are what a plugin sees. A change of the library that alters one of them fails here, so
// it is a decision about the plugins and not a side effect.

func getPinnedShapes() map[string]string {
	return map[string]string{
		"Address":            "struct{Scheme string; Value string}",
		"Backlink":           "struct{ResolvedLink domain.ResolvedLink; SourceRole domain.LinkRole}",
		"ByteRange":          "struct{Start int; End int}",
		"Connection":         "struct{A string; B string; AB *domain.Side; BA *domain.Side}",
		"FileStat":           "struct{Path string; Kind name.SourceKind; Size int64; ModTime time.Time; FileID name.FileID}",
		"Link":               "struct{Target domain.Address; Role domain.LinkRole; Type string; Why string; Label string; Marker domain.LinkMarker}",
		"LinkMarker":         "string",
		"LinkRole":           "string",
		"MoveResult":         "struct{From string; To string; IsMoved bool; Repaired []string; Dangling []string}",
		"Note":               "struct{FileStat name.FileStat; Title string; Type domain.NoteType; ID string; Frontmatter map[string]interface {}; " +
			"Headings []domain.Heading; Links []domain.Link; Body string; Problems []string; FrontmatterErr string}",
		"NoteRef":            "struct{Path string; Title string; ID string}",
		"NoteType":           "string",
		"RelatedNote":        "struct{From string; Role domain.LinkRole; Title string; Type string; Marker domain.LinkMarker}",
		"ResolvedLink":       "struct{Link domain.Link; From string; To string; ToVault domain.VaultID; IsAmbiguous bool}",
		"Selector":           "string",
		"Side":               "struct{Role domain.LinkRole; Type string; Why string; Label string; Marker domain.LinkMarker}",
		"SourceKind":         "string",
		"TextKind":           "string",
		"TextSegment":        "struct{Text string; Heading string; Selector domain.Selector}",
		"TextSourceProperty": "struct{Key string; Value string}",
		"TextVersion":        "struct{Path string; PluginID string; Key string; Source domain.TextVersionID; SourceProperty domain.TextSourceProperty; " +
			"Kind domain.TextKind; File string; InputHash string; TextHash string; CreateTime time.Time}",
		"TextVersionID":      "string",
		"Vault":              "struct{ID domain.VaultID; Name string; Path string}",
		"VaultID":            "string",
		"VaultProblem":       "struct{Path string; Kind domain.Check; Detail string; Target domain.Address; Candidates []string}",
		"VersionText":        "struct{version domain.TextVersion; Text string; segments []domain.TextSegment; offsets []int}",
	}
}

func describe(t reflect.Type) string {
	if t.Kind() != reflect.Struct {
		return t.Kind().String()
	}
	fields := make([]string, 0, t.NumField())
	for field := range t.Fields() {
		fields = append(fields, field.Name+" "+field.Type.String())
	}
	return "struct{" + strings.Join(fields, "; ") + "}"
}

func TestThePluginsSeeTheShapesOfTheNotesLibraryTheyAreBuiltAgainst(t *testing.T) {
	shapes := map[string]reflect.Type{
		"Vault":              reflect.TypeFor[domain.Vault](),
		"VaultID":            reflect.TypeFor[domain.VaultID](),
		"FileStat":           reflect.TypeFor[domain.FileStat](),
		"SourceKind":         reflect.TypeFor[domain.SourceKind](),
		"Note":               reflect.TypeFor[domain.Note](),
		"NoteRef":            reflect.TypeFor[domain.NoteRef](),
		"NoteType":           reflect.TypeFor[domain.NoteType](),
		"ResolvedLink":       reflect.TypeFor[domain.ResolvedLink](),
		"Link":               reflect.TypeFor[domain.Link](),
		"Address":            reflect.TypeFor[domain.Address](),
		"LinkRole":           reflect.TypeFor[domain.LinkRole](),
		"LinkMarker":         reflect.TypeFor[domain.LinkMarker](),
		"Connection":         reflect.TypeFor[domain.Connection](),
		"Side":               reflect.TypeFor[domain.Side](),
		"RelatedNote":        reflect.TypeFor[domain.RelatedNote](),
		"Backlink":           reflect.TypeFor[domain.Backlink](),
		"VaultProblem":       reflect.TypeFor[domain.VaultProblem](),
		"MoveResult":         reflect.TypeFor[domain.MoveResult](),
		"TextVersion":        reflect.TypeFor[domain.TextVersion](),
		"TextVersionID":      reflect.TypeFor[domain.TextVersionID](),
		"TextKind":           reflect.TypeFor[domain.TextKind](),
		"TextSourceProperty": reflect.TypeFor[domain.TextSourceProperty](),
		"TextSegment":        reflect.TypeFor[domain.TextSegment](),
		"Selector":           reflect.TypeFor[domain.Selector](),
		"VersionText":        reflect.TypeFor[domain.VersionText](),
		"ByteRange":          reflect.TypeFor[domain.ByteRange](),
	}
	names := make([]string, 0, len(shapes))
	for name := range shapes {
		names = append(names, name)
	}
	slices.Sort(names)
	pinned := getPinnedShapes()
	var report strings.Builder
	for _, name := range names {
		got := describe(shapes[name])
		fmt.Fprintf(&report, "\t%q: %q,\n", name, got)
		if want, ok := pinned[name]; !ok || want != got {
			t.Errorf("%s is %s, the plugins are built against %q", name, got, want)
		}
	}
	if t.Failed() {
		t.Logf("the shapes now:\n%s", report.String())
	}
}
