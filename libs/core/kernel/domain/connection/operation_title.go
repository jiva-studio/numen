package connection

import "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

// GetOperationTitle returns the title a plugin gives the operation, and its name where none does.
func GetOperationTitle(titles []manifest.Operation, operation string) string {
	for _, one := range titles {
		if one.Name == operation && one.Title != "" {
			return one.Title
		}
	}
	return operation
}
