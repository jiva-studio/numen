package load

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"gopkg.in/yaml.v3"
)

const CoreImportPath = "github.com/jiva-studio/numen/libs/core/kernel"
