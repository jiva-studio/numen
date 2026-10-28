package vectors

import "github.com/jiva-studio/numen/libs/core/port/embeddingport"

// DescribeModel returns the display name and recipe string of an embedder model.
func DescribeModel(embedder embeddingport.Embedder) (name, recipe string) {
	if embedder == nil {
		return "", ""
	}
	m := embedder.GetModel()
	return m.String(), m.Recipe()
}
