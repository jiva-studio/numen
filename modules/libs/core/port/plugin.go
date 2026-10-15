package port

import "errors"

var (
	// ErrPathTraversal is returned when a path escapes the active vault boundary.
	ErrPathTraversal = errors.New("path traversal outside active vault is prohibited")

	// ErrPermissionDenied is returned when a plugin lacks the required permission.
	ErrPermissionDenied = errors.New("plugin permission denied")
)
