// Package fakeserver is a local sync server that implements the contract of the protocol with the identity
// code the real server uses, for the tests of the plugin and the end-to-end suite: sign-in, the device chain
// of an account, the signed catalog, revisions with compare-and-swap on the parent and one seq per commit,
// and presigned blob URLs. It keeps everything in memory and nothing on disk.
package fakeserver
