package loggers

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

func getLoggerPackages() map[string]bool {
	return map[string]bool{
		"log": true, "slog": true, "logrus": true, "zap": true,
		"zerolog": true, "logr": true, "glog": true, "klog": true,
	}
}

func isLoggerPackage(to string) bool { return getLoggerPackages()[to[strings.LastIndex(to, "/")+1:]] }
