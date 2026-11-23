package note

// ClassifyWriteError splits the error of a write by whether the index followed it.
type ClassifyWriteError struct {
	follow IndexFollow
}

// NewClassifyWriteError returns the use case over the report of whether the index followed a write.
func NewClassifyWriteError(follow IndexFollow) *ClassifyWriteError {
	return &ClassifyWriteError{follow: follow}
}

// Execute splits the error of a write: when the write reached the vault and only the index did not follow, it
// returns true and no error; any other error is returned as it is.
func (u *ClassifyWriteError) Execute(err error) (isIndexOutdated bool, failure error) {
	if u.follow.IsIndexOutdated(err) {
		return true, nil
	}
	return false, err
}
