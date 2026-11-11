package machineclock

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func isClockRead(name string) bool { return name == "Now" || name == "Since" || name == "Until" }

func getClockReads(file *ast.File) (reads []string) {
	ast.Inspect(file, func(node ast.Node) bool {
		pos, is := node.(*ast.SelectorExpr)
		if !is {
			return true
		}
		from, is := pos.X.(*ast.Ident)
		if !is || from.Name != "time" {
			return true
		}
		if isClockRead(pos.Sel.Name) {
			reads = append(reads, "time."+pos.Sel.Name)
		}
		return true
	})
	return reads
}
