package transports

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

func listTransports() []string {
	return []string{
		"connectrpc.com",
		"database/sql",
		"net/http",
		"github.com/jiva-studio/numen/libs/core/protocol",
	}
}

func isTransport(to string) bool {
	for _, one := range listTransports() {
		if to == one || strings.HasPrefix(to, one+"/") {
			return true
		}
	}
	return false
}
