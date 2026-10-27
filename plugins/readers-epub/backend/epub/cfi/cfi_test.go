package cfi

import (
	"errors"
	"slices"
	"testing"
)

func TestParseReadsTheStepsAndTheOffset(t *testing.T) {
	for text, want := range map[string]CFI{
		"epubcfi(/6/4!/4/10/3:12)":                         {Package: []int{6, 4}, Document: []int{4, 10, 3}, Offset: 12},
		"epubcfi(/6/4[chap01ref]!/4[body01]/10[para05]/3)": {Package: []int{6, 4}, Document: []int{4, 10, 3}, Offset: -1},
		"epubcfi(/6/4!/4/2/1:3[xx^]y,z])":                  {Package: []int{6, 4}, Document: []int{4, 2, 1}, Offset: 3},
		"epubcfi(/6/4!/4/2,/1:3,/1:9)":                     {Package: []int{6, 4}, Document: []int{4, 2, 1}, Offset: 3},
		"epubcfi(/6/4,!/4/2/1:3,!/4/2/1:9)":                {Package: []int{6, 4}, Document: []int{4, 2, 1}, Offset: 3},
		"epubcfi(/6/4!/4/2/1:3~2.5@10:20)":                 {Package: []int{6, 4}, Document: []int{4, 2, 1}, Offset: 3},
		"epubcfi(/6/4!)":                                   {Package: []int{6, 4}, Offset: -1},
	} {
		got, err := Parse(text)
		if err != nil {
			t.Errorf("Parse(%s): %v", text, err)
			continue
		}
		if !slices.Equal(got.Package, want.Package) || !slices.Equal(got.Document, want.Document) || got.Offset != want.Offset {
			t.Errorf("Parse(%s) = %+v, want %+v", text, got, want)
		}
	}
}

func TestParseRejectsWhatIsNoCFIIntoADocument(t *testing.T) {
	for _, text := range []string{
		"", "/6/4!/4", "epubcfi(/6/4!/4", "epubcfi(/6/4/4/2)", "epubcfi(/6!/4/2)", "epubcfi(/6/4!/4!/2)",
		"epubcfi(/6/x!/4)", "epubcfi(/6/4!/4/2 junk)", "epubcfi(/6/4!:3)", "epubcfi(/6/4!/4,/2)",
	} {
		if got, err := Parse(text); !errors.Is(err, ErrSyntax) {
			t.Errorf("Parse(%q) = %+v, %v, want ErrSyntax", text, got, err)
		}
	}
}

func TestStringWritesTheCFIParseReads(t *testing.T) {
	for _, text := range []string{"epubcfi(/8/2!/4/2/1:0)", "epubcfi(/6/14!/4/6)", "epubcfi(/6/4!/4/2/3:120)"} {
		place, err := Parse(text)
		if err != nil {
			t.Fatal(err)
		}
		if got := place.String(); got != text {
			t.Errorf("String() = %s, want %s", got, text)
		}
	}
}
