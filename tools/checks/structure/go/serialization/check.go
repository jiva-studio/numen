package serialization

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

func getSerializationImports() map[string]bool {
	return map[string]bool{
		"encoding/json":              true,
		"encoding/xml":               true,
		"gopkg.in/yaml.v3":           true,
		"github.com/BurntSushi/toml": true,
		"google.golang.org/protobuf": true,
	}
}

func listSerializationTags() []string {
	return []string{"json:", "yaml:", "sql:", "db:", "xml:", "protobuf:"}
}
