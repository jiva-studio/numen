package claudecode

// GetSession is the session a conversation goes on with, for the tests standing
// outside this package.
func (a *Agent) GetSession(conversation string) string { return a.carries(conversation) }
