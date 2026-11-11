package drivingcalls

import (
	"go/ast"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"gopkg.in/yaml.v3"
)

func getPortFields(file *ast.File) []string {
	var found []string
	ast.Inspect(file, func(node ast.Node) bool {
		target, is := node.(*ast.StructType)
		if !is {
			return true
		}
		for _, one := range target.Fields.List {
			callType := one.Type
			if star, is := callType.(*ast.StarExpr); is {
				callType = star.X
			}
			pos, is := callType.(*ast.SelectorExpr)
			if !is {
				continue
			}
			if from, is := pos.X.(*ast.Ident); is && load.IsPortName(from.Name) {
				for _, name := range one.Names {
					found = append(found, name.Name)
				}
			}
		}
		return true
	})
	return found
}

func getFieldCalls(file *ast.File, fields []string) []string {
	var found []string
	ast.Inspect(file, func(node ast.Node) bool {
		call, is := node.(*ast.CallExpr)
		if !is {
			return true
		}
		method, is := call.Fun.(*ast.SelectorExpr)
		if !is {
			return true
		}
		field, is := method.X.(*ast.SelectorExpr)
		if is && slices.Contains(fields, field.Sel.Name) {
			found = append(found, field.Sel.Name+"."+method.Sel.Name)
		}
		return true
	})
	return found
}

func isDrivingAdapterFile(rel string) bool {
	return load.IsAmong([]string{"adapter/cli", "adapter/window", "adapter/plugin"}, filepath.ToSlash(filepath.Dir(rel)))
}

func depguardFiles(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", ".golangci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Linters struct {
			Settings struct {
				Depguard struct {
					Rules map[string]struct {
						Files []string `yaml:"files"`
					} `yaml:"rules"`
				} `yaml:"depguard"`
			} `yaml:"settings"`
		} `yaml:"linters"`
	}
	if err := yaml.Unmarshal(raw, &config); err != nil {
		t.Fatal(err)
	}
	return config.Linters.Settings.Depguard.Rules["driving-adapter"].Files
}

func getUncovered(patterns []string) []string {
	var found []string
	for one := range load.GetDriving() {
		if !slices.ContainsFunc(patterns, func(pattern string) bool {
			target := strings.TrimSuffix(strings.TrimPrefix(pattern, "**/"), "/**")
			return !strings.HasPrefix(pattern, "!") && (one == target || strings.HasPrefix(one, target+"/"))
		}) {
			found = append(found, one)
		}
	}
	slices.Sort(found)
	return found
}
