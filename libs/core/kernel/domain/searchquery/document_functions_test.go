package searchquery_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

func getZone(t *testing.T, name string) *time.Location {
	t.Helper()
	zone, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return zone
}

func TestParseDateAcceptsTheDocumentedFormsAndRejectsOthers(t *testing.T) {
	kolkata := getZone(t, "Asia/Kolkata")
	cases := map[string]string{
		"2026-05-01":                     "2026-04-30T18:30:00Z",
		"2026-05-01T10:15":               "2026-05-01T04:45:00Z",
		"2026-05-01 10:15:30":            "2026-05-01T04:45:30Z",
		"2026-05-01T10:15:30.25Z":        "2026-05-01T10:15:30.25Z",
		"2026-05-01T10:15+03:00":         "2026-05-01T07:15:00Z",
		"2026-05-01T10:15:30.123456789Z": "2026-05-01T10:15:30.123456789Z",
	}
	for text, want := range cases {
		got, err := searchquery.ParseDate(text, kolkata)
		if err != nil {
			t.Errorf("%s: %v", text, err)
			continue
		}
		if got.UTC().Format(time.RFC3339Nano) != want {
			t.Errorf("%s is %s, want %s", text, got.UTC().Format(time.RFC3339Nano), want)
		}
	}
	for _, text := range []string{"2026-02-30", "2026-1-5", "01.05.2026", "2026-05-01T1:00", "2026-05-01T10", ""} {
		if _, err := searchquery.ParseDate(text, kolkata); !errors.Is(err, searchquery.ErrInvalidDate) {
			t.Errorf("%q returned %v, want ErrInvalidDate", text, err)
		}
	}
}

func text(value string) searchquery.Value {
	return searchquery.Value{Kind: searchquery.ValueString, Text: value}
}

func integer(value int64) searchquery.Value {
	return searchquery.Value{Kind: searchquery.ValueInt, Int: value}
}

func double(value float64) searchquery.Value {
	return searchquery.Value{Kind: searchquery.ValueDouble, Double: value}
}

func instant(value time.Time) searchquery.Value {
	return searchquery.Value{Kind: searchquery.ValueTime, Time: value}
}

func storedText(value string) searchquery.PropertyValue {
	return searchquery.PropertyValue{Kind: searchquery.PropertyString, Text: value}
}

func storedList(items ...searchquery.PropertyValue) searchquery.PropertyValue {
	return searchquery.PropertyValue{Kind: searchquery.PropertyList, Items: items}
}

func TestIsPropertyMatchFollowsTheStoredType(t *testing.T) {
	zone := getZone(t, "Europe/Moscow")
	stored1 := searchquery.PropertyValue{Kind: searchquery.PropertyInt, Int: 1}
	storedNull := searchquery.PropertyValue{Kind: searchquery.PropertyNull}
	storedTrue := searchquery.PropertyValue{Kind: searchquery.PropertyBool, Bool: true}
	storedMap := searchquery.PropertyValue{Kind: searchquery.PropertyMap}
	may := time.Date(2026, 5, 1, 0, 0, 0, 0, zone)
	null := searchquery.Value{Kind: searchquery.ValueNull}
	yes := searchquery.Value{Kind: searchquery.ValueBool, Bool: true}
	cases := []struct {
		name       string
		value      searchquery.PropertyValue
		comparison searchquery.Comparison
		operands   []searchquery.Value
		want       bool
	}{
		{"1 == 1.0", stored1, searchquery.ComparisonEqual, []searchquery.Value{double(1)}, true},
		{"1 < 1.5", stored1, searchquery.ComparisonLess, []searchquery.Value{double(1.5)}, true},
		{`"4" == 4`, storedText("4"), searchquery.ComparisonEqual, []searchquery.Value{integer(4)}, false},
		{`4 == "4"`, searchquery.PropertyValue{Kind: searchquery.PropertyInt, Int: 4}, searchquery.ComparisonEqual, []searchquery.Value{text("4")}, false},
		{"string ==", storedText("done"), searchquery.ComparisonEqual, []searchquery.Value{text("done")}, true},
		{"string order", storedText("b"), searchquery.ComparisonGreater, []searchquery.Value{text("a")}, true},
		{"date string == date", storedText("2026-05-01"), searchquery.ComparisonEqual, []searchquery.Value{instant(may)}, true},
		{"date string < date", storedText("2026-04-30T23:00"), searchquery.ComparisonLess, []searchquery.Value{instant(may)}, true},
		{"word < date", storedText("soon"), searchquery.ComparisonLess, []searchquery.Value{instant(may)}, false},
		{"bool ==", storedTrue, searchquery.ComparisonEqual, []searchquery.Value{yes}, true},
		{"bool <", storedTrue, searchquery.ComparisonLess, []searchquery.Value{yes}, false},
		{"null == null", storedNull, searchquery.ComparisonEqual, []searchquery.Value{null}, true},
		{"null in property", storedNull, searchquery.ComparisonContains, []searchquery.Value{null}, false},
		{"null in [null]", storedNull, searchquery.ComparisonIn, []searchquery.Value{text("a"), null}, true},
		{"list == item", storedList(storedText("a")), searchquery.ComparisonEqual, []searchquery.Value{text("a")}, false},
		{"item in list", storedList(storedText("a"), storedText("b")), searchquery.ComparisonContains, []searchquery.Value{text("b")}, true},
		{"item in scalar", storedText("b"), searchquery.ComparisonContains, []searchquery.Value{text("b")}, true},
		{"list in [literals]", storedList(storedText("a"), storedText("b")), searchquery.ComparisonIn, []searchquery.Value{text("c"), text("b")}, true},
		{"scalar in [literals]", stored1, searchquery.ComparisonIn, []searchquery.Value{integer(2), integer(1)}, true},
		{"map ==", storedMap, searchquery.ComparisonEqual, []searchquery.Value{text("a")}, false},
		{"map in", storedMap, searchquery.ComparisonContains, []searchquery.Value{text("a")}, false},
		{"present", storedNull, searchquery.ComparisonPresent, nil, true},
	}
	for _, c := range cases {
		if got := searchquery.IsPropertyMatch(c.value, c.comparison, c.operands, zone); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

func TestIsHeadingMatchComparesTheWholeHeadingIgnoringCase(t *testing.T) {
	cases := []struct {
		heading, text string
		want          bool
	}{
		{"  Summary ", "summary", true},
		{"Summary of the week", "Summary", false},
		{"Weekly summary", "Summary", false},
		{"Итоги", "итоги", true},
	}
	for _, c := range cases {
		if got := searchquery.IsHeadingMatch(c.heading, c.text); got != c.want {
			t.Errorf("IsHeadingMatch(%q, %q) is %v, want %v", c.heading, c.text, got, c.want)
		}
	}
}

func newPropertyEqual(key string, value searchquery.Value) searchquery.RawNode {
	return searchquery.RawNode{Operator: searchquery.OperatorLeaf, Leaf: searchquery.RawLeaf{
		Kind: searchquery.LeafSet, Function: searchquery.FunctionProperty, Key: key,
		Comparison: searchquery.ComparisonEqual, Operands: []searchquery.Value{value}, Position: searchquery.StartPosition(),
	}}
}

func newHasProperty(key string) searchquery.RawNode {
	return searchquery.RawNode{Operator: searchquery.OperatorLeaf, Leaf: searchquery.RawLeaf{
		Kind: searchquery.LeafSet, Function: searchquery.FunctionHasProperty, Key: key,
		Comparison: searchquery.ComparisonPresent, Position: searchquery.StartPosition(),
	}}
}

func newCompare(attribute searchquery.Attribute, comparison searchquery.Comparison, operands ...searchquery.Value) searchquery.RawNode {
	return searchquery.RawNode{Operator: searchquery.OperatorLeaf, Leaf: searchquery.RawLeaf{
		Kind: searchquery.LeafCompare, Attribute: attribute, Comparison: comparison, Operands: operands, Position: searchquery.StartPosition(),
	}}
}

func TestANegatedPropertyEqualMatchesADocumentOutsideTheSet(t *testing.T) {
	q := normalize(t, newAnd(newKind("note", "pdf"), newNot(newPropertyEqual("status", text("done")))))
	if len(q.Sets) != 1 {
		t.Fatalf("sets %+v, want one", q.Sets)
	}
	d := getNote("a.md")
	d.IsInSet = []bool{false}
	if !searchquery.IsDocumentMatch(q.Root, d) {
		t.Error("a note without the key does not match property(status) != done")
	}
	d.IsInSet = []bool{true}
	if searchquery.IsDocumentMatch(q.Root, d) {
		t.Error("a note with status done matches property(status) != done")
	}
	if !searchquery.IsDocumentMatch(q.Root, getPDF("b.pdf")) {
		t.Error("a pdf does not match property(status) != done")
	}
}

func TestEqualSetConditionsShareOneIndex(t *testing.T) {
	q := normalize(t, newOr(newHasProperty("a"), newAnd(newHasProperty("a"), newPropertyEqual("a", integer(1)))))
	if len(q.Sets) != 2 {
		t.Errorf("sets %+v, want two", q.Sets)
	}
}

func TestADateOperandIsReadInTheZoneOfTheQuery(t *testing.T) {
	zone := getZone(t, "Europe/Moscow")
	raw := newCompare(searchquery.AttributeModified, searchquery.ComparisonGreaterOrEqual, searchquery.Value{Kind: searchquery.ValueDate, Text: "2026-01-01"})
	q, err := searchquery.Normalize(raw, searchquery.DefaultLimits(), zone)
	if err != nil {
		t.Fatal(err)
	}
	if got := q.Root.Leaf.Operands[0]; got.Kind != searchquery.ValueTime || !got.Time.Equal(time.Date(2025, 12, 31, 21, 0, 0, 0, time.UTC)) {
		t.Errorf("the operand is %+v, want midnight in Moscow", got)
	}
	d := getNote("a.md")
	d.ModifiedAt = time.Date(2025, 12, 31, 23, 59, 0, 0, zone)
	if searchquery.IsDocumentMatch(q.Root, d) {
		t.Error("a note changed at 23:59 local on 2025-12-31 matches")
	}
	d.ModifiedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, zone)
	if !searchquery.IsDocumentMatch(q.Root, d) {
		t.Error("a note changed at local midnight does not match")
	}
}

func TestAnInvalidDateAndAnEmptyKeyAreProblems(t *testing.T) {
	bad := newCompare(searchquery.AttributeModified, searchquery.ComparisonLess, searchquery.Value{Kind: searchquery.ValueDate, Text: "2026-13-01", Position: searchquery.Position{Line: 1, Column: 17}})
	_, err := searchquery.Normalize(newAnd(bad, newHasProperty(" ")), searchquery.DefaultLimits(), time.UTC)
	parseError, ok := errors.AsType[*searchquery.ParseError](err)
	if !ok || len(parseError.Problems) != 2 {
		t.Fatalf("returned %v, want two problems", err)
	}
	if parseError.Problems[0].Position.Column != 17 {
		t.Errorf("the date problem is at %+v, want column 17", parseError.Problems[0].Position)
	}
}

func TestTitleReadsTheNoteTitleElseTheFileName(t *testing.T) {
	q := normalize(t, newCompare(searchquery.AttributeTitle, searchquery.ComparisonEqual, text("Gita.v2")))
	pdf := getPDF("books/Gita.v2.pdf")
	pdf.Title = searchquery.GetTitle(pdf.Path, "")
	if !searchquery.IsDocumentMatch(q.Root, pdf) {
		t.Errorf("title of %s is %q, want Gita.v2", pdf.Path, pdf.Title)
	}
	note := getNote("notes/Gita.v2.md")
	note.Title = searchquery.GetTitle(note.Path, "Bhagavad Gita")
	if searchquery.IsDocumentMatch(q.Root, note) {
		t.Error("the file name of a note won over its title")
	}
}

func TestAPositiveSetLeafNarrowsTheScopeOfAMeaning(t *testing.T) {
	q := normalize(t, newAnd(newMeaning("a"), newHasProperty("src")))
	scopes := searchquery.GetScopes(q, nil)
	if len(scopes) != 1 || !slices.Equal(scopes[0].SetIndexes, []int{0}) {
		t.Errorf("scopes %+v, want the set of hasProperty", scopes)
	}
	if scope := searchquery.GetScopes(normalize(t, newAnd(newMeaning("a"), newNot(newHasProperty("src")))), nil)[0]; len(scope.SetIndexes) != 0 {
		t.Errorf("a negated set narrows the scope: %+v", scope)
	}
}

func TestTheCatalogDescribesEveryDocumentFunction(t *testing.T) {
	catalog := searchquery.GetBuiltinCatalog()
	for _, name := range []string{
		searchquery.VariableTitle, searchquery.VariableNoteType, searchquery.VariableModified, searchquery.VariableSize,
		searchquery.FunctionDate, searchquery.FunctionProperty, searchquery.FunctionHasProperty, searchquery.FunctionHasHeading,
	} {
		f, ok := catalog.FindFunction(name)
		if !ok || f.Description == "" || f.Example == "" || f.Level != searchquery.LevelDocument {
			t.Errorf("%s is %+v", name, f)
		}
	}
}

func TestAnIntegerBeyondTheExactRangeOfADoubleComparesExactly(t *testing.T) {
	const beyond = int64(9007199254740993)
	exact := float64(1 << 53)
	value := searchquery.PropertyValue{Kind: searchquery.PropertyInt, Int: beyond}
	cases := []struct {
		name       string
		comparison searchquery.Comparison
		operand    searchquery.Value
		want       bool
	}{
		{"is not equal to the double below it", searchquery.ComparisonEqual, double(exact), false},
		{"is greater than the double below it", searchquery.ComparisonGreater, double(exact), true},
		{"is not in the list of the double below it", searchquery.ComparisonIn, double(exact), false},
		{"is equal to its own integer", searchquery.ComparisonEqual, integer(beyond), true},
		{"is less than a larger double", searchquery.ComparisonLess, double(exact * 2), true},
		{"is greater than a fraction", searchquery.ComparisonGreater, double(0.5), true},
	}
	for _, c := range cases {
		if got := searchquery.IsPropertyMatch(value, c.comparison, []searchquery.Value{c.operand}, time.UTC); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	floating := searchquery.PropertyValue{Kind: searchquery.PropertyDouble, Double: exact}
	if searchquery.IsPropertyMatch(floating, searchquery.ComparisonEqual, []searchquery.Value{integer(beyond)}, time.UTC) {
		t.Error("the double below the integer is equal to it")
	}
	if !searchquery.IsPropertyMatch(floating, searchquery.ComparisonLess, []searchquery.Value{integer(beyond)}, time.UTC) {
		t.Error("the double below the integer is not less than it")
	}
}
