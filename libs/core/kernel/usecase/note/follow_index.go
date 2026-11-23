package note

// FollowIndex settles the outcome of a write by whether the index followed it.
type FollowIndex struct {
	follow IndexFollow
}

// NewFollowIndex returns the use case over the report of whether the index followed a write.
func NewFollowIndex(follow IndexFollow) *FollowIndex {
	return &FollowIndex{follow: follow}
}

// Settle reports whether the index did not follow the write that ended in err. A write that reached the vault
// and not the index has happened, so its error is dropped and every other error is returned.
func (u *FollowIndex) Settle(err error) (bool, error) {
	if u.follow.IsIndexOutdated(err) {
		return true, nil
	}
	return false, err
}
