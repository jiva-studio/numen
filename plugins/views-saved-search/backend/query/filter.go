package query

import "slices"

// Filter narrows the searches the host runs for a query. It is taken from the top-level conjuncts of the
// root only; every kind and path term is evaluated again for each candidate. Kinds are the kinds of source
// the host searches, "note" or "file", every kind when empty. Prefixes are the folders to search, one search
// each, the whole vault when empty. IsEveryKindExcluded is set when the kind terms hold for no kind of file.
type Filter struct {
	Kinds               []string
	Prefixes            []string
	IsEveryKindExcluded bool
}

func getFilter(root Node) Filter {
	conjuncts := []Node{root}
	if root.Operator == OperatorAnd {
		conjuncts = root.Children
	}
	kinds, isEveryKindExcluded := getKinds(conjuncts)
	return Filter{Kinds: kinds, Prefixes: getPrefixes(conjuncts), IsEveryKindExcluded: isEveryKindExcluded}
}

func getKinds(conjuncts []Node) ([]string, bool) {
	allowed := listKinds()
	isNarrowed := false
	for _, node := range conjuncts {
		if node.Operator != OperatorLeaf || node.Term.Kind != TermKind {
			continue
		}
		isNarrowed = true
		allowed = slices.DeleteFunc(allowed, func(kind string) bool { return slices.Contains(node.Term.Kinds, kind) == node.IsNegated })
	}
	if !isNarrowed {
		return nil, false
	}
	if len(allowed) == 0 {
		return nil, true
	}
	var hostKinds []string
	for _, kind := range allowed {
		if hostKind := getHostKind(kind); !slices.Contains(hostKinds, hostKind) {
			hostKinds = append(hostKinds, hostKind)
		}
	}
	if len(hostKinds) > 1 {
		return nil, false
	}
	return hostKinds, false
}

func getPrefixes(conjuncts []Node) []string {
	for _, node := range conjuncts {
		if isPathTerm(node) {
			return []string{node.Term.Prefix}
		}
		if node.Operator == OperatorOr && !slices.ContainsFunc(node.Children, func(child Node) bool { return !isPathTerm(child) }) {
			var prefixes []string
			for _, child := range node.Children {
				if !slices.Contains(prefixes, child.Term.Prefix) {
					prefixes = append(prefixes, child.Term.Prefix)
				}
			}
			return prefixes
		}
	}
	return nil
}

func isPathTerm(node Node) bool {
	return node.Operator == OperatorLeaf && node.Term.Kind == TermPath && !node.IsNegated
}
