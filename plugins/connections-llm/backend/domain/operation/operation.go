// Package operation names what a connection is asked to do.
package operation

// The operations the plugin carries out. List reads the models of the service.
const (
	Complete = "complete"
	Chat     = "chat"
	Embed    = "embed"
	List     = "list"
)

// IsText reports whether the operation gives text: a completion or a chat.
func IsText(operation string) bool {
	return operation == Complete || operation == Chat
}
