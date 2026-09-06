package review_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"
	"testing"
)

// placing is every arithmetic that places a card's day: the session that hands
// the cards out, and the projection drawn beside a control. Both go through
// Places, and that is the whole of the list.
//
// A function is one arithmetic however many times its body asks — the
// projection places a card it has begun and a card it has answered again, and
// they are one step. A third function here is a third arithmetic, kept in step
// with these two by tests and drifting the day one of them is touched.
var placing = []string{"History.Replay", "Simulation.step"}

// A day is chosen in one function, and a caller wanting one comes to it. The
// walk is over this package's own files, because Places is where a day is
// decided and the deciding is all here.
func TestTheSessionAndTheProjectionAreWhatPlacesADay(t *testing.T) {
	held, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var calls []string
	read, declared := 0, 0
	for _, one := range held {
		if one.IsDir() || !strings.HasSuffix(one.Name(), ".go") ||
			strings.HasSuffix(one.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), one.Name(), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		read++
		declared += declaring(file, "Places")
		calls = append(calls, calling(file, "Places")...)
	}

	// A walk that read none of the package, or one that did not find the
	// function the rule is about, is a rule checked against nothing.
	if read < 10 {
		t.Fatalf("%d files of this package read: the walk is not reading it", read)
	}
	if declared != 1 {
		t.Fatalf("%d functions named Places are declared here: the walk is reading something else", declared)
	}

	slices.Sort(calls)
	calls = slices.Compact(calls)
	for _, one := range calls {
		if !slices.Contains(placing, one) {
			t.Errorf("%s places a card's day: one function places it and everything else comes to that one", one)
		}
	}
	for _, one := range placing {
		if !slices.Contains(calls, one) {
			t.Errorf("%s is named here as an arithmetic placing a day and places none", one)
		}
	}
}

// Put over sources written for it, the rule has to name the function a call
// stands in and pass over everything that only looks like one.
func TestWhatThePlacingRuleRefuses(t *testing.T) {
	for _, one := range []struct {
		why, src string
		calls    []string
	}{{
		why:   "a method calling it is named by its receiver's type",
		src:   "package review\nfunc (s Simulation) step() { p.Places(on, at, due) }\n",
		calls: []string{"Simulation.step"},
	}, {
		why:   "a function calling it is named on its own",
		src:   "package review\nfunc drawn() { p.Places(on, at, due) }\n",
		calls: []string{"drawn"},
	}, {
		why:   "a body asking twice is one arithmetic",
		src:   "package review\nfunc (s Simulation) step() { p.Places(a, b, c); p.Places(d, e, f) }\n",
		calls: []string{"Simulation.step"},
	}, {
		why:   "a call inside a literal is the function holding the literal",
		src:   "package review\nfunc (h History) Replay() { each(func() { one.Preset.Places(on, at, due) }) }\n",
		calls: []string{"History.Replay"},
	}, {
		why:   "a pointer receiver is named by the type it is of",
		src:   "package review\nfunc (s *Simulation) step() { p.Places(on, at, due) }\n",
		calls: []string{"Simulation.step"},
	}, {
		why: "the declaration of Places is not a call of it",
		src: "package review\nfunc (p Preset) Places(s *DueByDay) { p.lands(s) }\n",
	}, {
		why: "a comment naming it is not a call",
		src: "package review\n// Places is where the day is chosen.\nfunc drawn() { p.lands(on) }\n",
	}, {
		why: "another name is another function",
		src: "package review\nfunc drawn() { p.Placed(on, at, due) }\n",
	}, {
		why: "naming it without calling it is not a call",
		src: "package review\nfunc drawn() { each(p.Places) }\n",
	}} {
		file, err := parser.ParseFile(token.NewFileSet(), "one.go", one.src, 0)
		if err != nil {
			t.Fatalf("%s: %v", one.why, err)
		}
		if got := calling(file, "Places"); !slices.Equal(got, one.calls) {
			t.Errorf("%s: %v, and the rule reads %v", one.why, one.calls, got)
		}
	}
}

// calling is every function of a file whose body calls this one, named by the
// type it is a method of and by its own name. A function calling it more than
// once is named once: what the rule counts is arithmetics and not call sites.
func calling(file *ast.File, name string) []string {
	var out []string
	for _, decl := range file.Decls {
		at, is := decl.(*ast.FuncDecl)
		if !is || at.Body == nil {
			continue
		}
		var calls bool
		ast.Inspect(at.Body, func(node ast.Node) bool {
			call, is := node.(*ast.CallExpr)
			if !is {
				return true
			}
			if to, is := call.Fun.(*ast.SelectorExpr); is && to.Sel.Name == name {
				calls = true
			}
			return true
		})
		if calls {
			out = append(out, named(at))
		}
	}
	return out
}

// declaring is how many functions of a file are declared under this name.
func declaring(file *ast.File, name string) int {
	out := 0
	for _, decl := range file.Decls {
		if at, is := decl.(*ast.FuncDecl); is && at.Name.Name == name {
			out++
		}
	}
	return out
}

// named is what a function is called, a method by the type it is of.
func named(at *ast.FuncDecl) string {
	if at.Recv == nil || len(at.Recv.List) == 0 {
		return at.Name.Name
	}
	of := at.Recv.List[0].Type
	if star, is := of.(*ast.StarExpr); is {
		of = star.X
	}
	if held, is := of.(*ast.Ident); is {
		return held.Name + "." + at.Name.Name
	}
	return at.Name.Name
}
