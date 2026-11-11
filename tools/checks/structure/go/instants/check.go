package instants

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

func getInstants(file *ast.File) []load.Identity {
	return load.GetFieldsNamed(file, "Time")
}
