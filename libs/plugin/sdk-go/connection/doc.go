// Package connection is how a plugin takes part in the connections of the core: a provider carries out
// the operations of the connection types its manifest declares, and a consumer asks the core to carry
// out an operation on the rows of its own settings.
//
// The core knows an operation by its name only. The payload and the frames are bytes of the contract
// the operation belongs to; the packages under contracts give each contract a typed client and a typed
// provider adapter over this package. A provider mounts its operations with Mount. A consumer calls a
// Client, which names a connection only to prefer one of its rows and never receives a key.
package connection
