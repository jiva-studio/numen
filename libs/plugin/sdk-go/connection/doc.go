// Package connection is how a plugin takes part in the connections of the core: a provider carries out
// the operations of the connection types its manifest declares, and a consumer asks the core to carry
// out an operation on the rows of its own settings.
//
// A provider fills the roles of a Provider and mounts it with Mount. A consumer calls a Client, which
// never names a connection and never receives a key.
package connection
