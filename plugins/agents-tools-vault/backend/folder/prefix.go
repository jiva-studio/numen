// Package folder reads the folder an agent names relative to the vault root.
package folder

import "strings"

// GetPrefix returns the folder with its leading "/" and "./" and its trailing "/" removed and one "/"
// added at the end; the vault root, as "", "." or "/", gives "".
func GetPrefix(name string) string {
	for {
		next := strings.TrimPrefix(strings.TrimLeft(name, "/"), "./")
		if next == name {
			break
		}
		name = next
	}
	name = strings.TrimRight(name, "/")
	if name == "" || name == "." {
		return ""
	}
	return name + "/"
}
