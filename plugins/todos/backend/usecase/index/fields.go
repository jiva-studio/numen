package index

import (
	"slices"
	"strings"
)

// maxFields is how many projects and how many contexts ListFields returns.
const maxFields = 200

// FieldCount is a project or a context and the number of tasks that carry it.
type FieldCount struct {
	Name  string
	Count int
}

// Fields is the projects and the contexts of the tasks.
type Fields struct {
	Projects []FieldCount
	Contexts []FieldCount
}

// ListFields returns the projects and the contexts of the tasks, the most used first and by name
// among the same count.
func (i *Index) ListFields() Fields {
	i.mu.Lock()
	defer i.mu.Unlock()
	return Fields{Projects: rank(i.projects), Contexts: rank(i.contexts)}
}

func rank(counts map[string]int) []FieldCount {
	ranked := make([]FieldCount, 0, len(counts))
	for name, count := range counts {
		ranked = append(ranked, FieldCount{Name: name, Count: count})
	}
	slices.SortFunc(ranked, func(a, b FieldCount) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		return strings.Compare(a.Name, b.Name)
	})
	if len(ranked) > maxFields {
		ranked = ranked[:maxFields]
	}
	return ranked
}
