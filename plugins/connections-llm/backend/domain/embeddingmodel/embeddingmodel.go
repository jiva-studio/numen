// Package embeddingmodel holds what the plugin knows of the embedding models of the services it
// serves, from a table, with no request.
package embeddingmodel

import (
	"slices"
	"strings"
)

// PoolingCLS is how the table says the output of a model becomes a vector.
const PoolingCLS = "cls"

// Profile is the shape of the vectors of a model.
type Profile struct {
	Dimensions int
	MaxTokens  int
	Pooling    string
}

type entry struct {
	ids     []string
	profile Profile
}

func listEntries() []entry {
	return []entry{
		{[]string{"openai/text-embedding-3-small", "text-embedding-3-small"}, Profile{1536, 8191, PoolingCLS}},
		{[]string{"openai/text-embedding-3-large", "text-embedding-3-large"}, Profile{3072, 8191, PoolingCLS}},
		{[]string{"openai/text-embedding-ada-002", "text-embedding-ada-002"}, Profile{1536, 8191, PoolingCLS}},
		{[]string{"baai/bge-m3"}, Profile{1024, 8192, PoolingCLS}},
		{[]string{"qwen/qwen3-embedding-8b"}, Profile{4096, 32768, PoolingCLS}},
	}
}

// Find returns the profile of the model, ignoring case and surrounding space, or false for a model
// that is not in the table.
func Find(model string) (Profile, bool) {
	name := strings.ToLower(strings.TrimSpace(model))
	for _, one := range listEntries() {
		if slices.Contains(one.ids, name) {
			return one.profile, true
		}
	}
	return Profile{}, false
}
