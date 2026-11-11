package load

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"golang.org/x/mod/modfile"
)

// PluginsPrefix is where the plugins are compiled from. A package under it is a
// plugin, and a module under it is one too.
const PluginsPrefix = "github.com/jiva-studio/numen/plugins/"

func IsPlugin(importPath string) bool {
	return strings.HasPrefix(importPath, PluginsPrefix)
}
