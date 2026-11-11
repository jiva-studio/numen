package serialization

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
