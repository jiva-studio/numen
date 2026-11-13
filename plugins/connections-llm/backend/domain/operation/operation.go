// Package operation names what a connection is asked to do, as the limits of a request tell them apart.
package operation

// The operations the plugin carries out. List reads the models of the service.
const (
	Complete = "complete"
	Chat     = "chat"
	Embed    = "embed"
	List     = "list"
)
