package pluginmanifest

import (
	"github.com/jiva-studio/numen/libs/plugin/catalog/searchfunction"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

func convertSearchFunctions(pluginID string, declared searchfunction.Set) []searchquery.Function {
	out := make([]searchquery.Function, 0, len(declared.Functions))
	for _, one := range declared.Functions {
		f := searchquery.Function{
			Name: one.Name, Result: searchquery.TypeBool, Level: searchquery.LevelDocument, PluginID: pluginID,
			Description: one.Description, Example: one.Example, Args: []searchquery.ValueType{}, ArgNames: []string{},
		}
		for _, arg := range one.Args {
			f.Args = append(f.Args, getArgumentType(arg.Type))
			f.ArgNames = append(f.ArgNames, arg.Name)
		}
		out = append(out, f)
	}
	return out
}

func getArgumentType(t searchfunction.ArgType) searchquery.ValueType {
	switch t {
	case searchfunction.ArgInt:
		return searchquery.TypeInt
	case searchfunction.ArgBool:
		return searchquery.TypeBool
	case searchfunction.ArgString:
		return searchquery.TypeString
	}
	return searchquery.TypeString
}
