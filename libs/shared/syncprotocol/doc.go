// Package syncprotocol is the protocol between Numen Sync and the sync server: the sign-in service, the
// bytes a device signs to sign in, to commit a revision and to append a catalog entry, the checks a client makes, the errors of the server and the version header with its two interceptors. It holds
// values only, so a use case may import it; the clients of the sign-in, catalog, revision, blob and account services are in
// the package client.
//
// The plugin and the server share it. client.SignInClient is the Go client that hides the RPC library from the plugin.
package syncprotocol
