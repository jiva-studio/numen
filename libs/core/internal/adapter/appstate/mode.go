package appstate

import "io/fs"

// The state is this user's alone: a file appstate writes is readable by its
// owner only, and so is a folder it makes.
const (
	privateFileMode fs.FileMode = 0o600
	privateDirMode  fs.FileMode = 0o700
)
