package instants

import (
	"go/ast"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func getInstants(file *ast.File) []load.Identity {
	return load.GetFieldsNamed(file, "Time")
}
