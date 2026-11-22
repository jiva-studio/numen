// Package syncprotocol is the protocol between Numen Sync and the sync server: the sign-in service, the
// bytes a device signs to sign in, the check a client makes of a challenge, and the version header with its
// two interceptors.
//
// The plugin and the server share it. SignInClient is the Go client that hides the RPC library from the plugin,
// and fakeserver is a local server for tests.
package syncprotocol
