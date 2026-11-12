package settings

import "strings"

// AppOwner is the owner of the settings of the shell. Every other first-level key of the file
// other than the version is the id of a plugin.
const AppOwner = "app"

// HostFieldEnabled is the field of the object of a plugin that the host keeps. A plugin that has no
// stored value for it is enabled, unless its manifest keeps it off by default.
const HostFieldEnabled = "enabled"

// HostFieldDiscover is the field of the object of a plugin that holds, by the key of each setting of rows that
// declares discover, whether it uses every connection that serves its operations while it holds no row.
const HostFieldDiscover = "discover"

// ConnectionsKey is the first-level key of the connections section.
const ConnectionsKey = "connections"

// Values is the value in force of every declared setting as JSON, by its path joined with
// JoinPath. A setting with neither a stored value nor a default is left out.
type Values map[string]string

const pathSeparator = "\x00"

// JoinPath is the key of a setting in Values.
func JoinPath(path []string) string { return strings.Join(path, pathSeparator) }

// SplitPath is the path a key of Values stands for.
func SplitPath(key string) []string { return strings.Split(key, pathSeparator) }

// IsPluginSetting reports whether path names a setting of a plugin: its owner and its key.
func IsPluginSetting(path []string) bool {
	return len(path) == 2 && path[0] != AppOwner
}

// Stored is the content of the settings file.
type Stored interface {
	// GetValue returns the value in force at a path: the stored value, else the default its owner
	// declares. The first element of the path is the owner.
	GetValue(path ...string) (any, bool)
	// GetConnectionsJSON is the connections section as JSON, and empty where the file holds no connection.
	GetConnectionsJSON() (string, error)
}
