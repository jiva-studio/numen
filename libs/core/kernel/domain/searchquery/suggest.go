package searchquery

// SuggestName returns the catalog name closest to name by Damerau-Levenshtein distance (optimal string
// alignment), when the distance is at most 2 and below half the length of name. Ties go to the name first
// in the catalog.
func SuggestName(name string, catalog Catalog) (string, bool) {
	runes := []rune(name)
	best, bestDistance := "", maxSuggestionDistance+1
	for _, f := range catalog.Functions {
		d := getDistance(runes, []rune(f.Name))
		if d < bestDistance {
			best, bestDistance = f.Name, d
		}
	}
	if bestDistance > maxSuggestionDistance || bestDistance*2 >= len(runes) {
		return "", false
	}
	return best, true
}

const maxSuggestionDistance = 2

func getDistance(a, b []rune) int {
	rows := make([][]int, len(a)+1)
	for i := range rows {
		rows[i] = make([]int, len(b)+1)
		rows[i][0] = i
	}
	for j := range rows[0] {
		rows[0][j] = j
	}
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			rows[i][j] = getCellDistance(a, b, rows, i, j)
		}
	}
	return rows[len(a)][len(b)]
}

func getCellDistance(a, b []rune, rows [][]int, i, j int) int {
	cost := 1
	if a[i-1] == b[j-1] {
		cost = 0
	}
	d := min(rows[i-1][j]+1, rows[i][j-1]+1, rows[i-1][j-1]+cost)
	if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
		d = min(d, rows[i-2][j-2]+1)
	}
	return d
}
