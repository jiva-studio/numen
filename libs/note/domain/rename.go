package domain

// GetNameAfterMove decides what a note is called once its file moves from one
// path to another. A note named by its title keeps the title only while its
// filed name matches the name of the file; a note without a title takes the new
// file name. isRenamed is false when the file name is unchanged, or when a
// titled note was already filed under a different name than its title gives.
func GetNameAfterMove(from, to, title string, isNamed bool) (display string, isRenamed bool) {
	name := Basename(to)
	if name == Basename(from) {
		return "", false
	}
	if !isNamed {
		return name, true
	}
	if filed, _ := GetReducedFilename(title); FoldName(filed) != FoldName(name) {
		return "", false
	}
	return title, true
}
