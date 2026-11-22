// Package connection holds the use cases that create, change, list and delete the connections the
// core sends requests to for the plugins, and list the kinds of connection the plugins provide.
//
// It also carries out the operations plugins ask for on the connections in their rows (Calls), and lists the
// values a row field of a connection can take, from the lists kept of what the provider answered (Options).
package connection
