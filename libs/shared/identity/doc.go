// Package identity is the key model of a Numen account: the 12 recovery words, the keys derived from
// them, the account keys wrapped for a recovery key or a device, the signed chain of devices and the
// record one device keeps.
//
// The core and the sync server share it. It uses only the standard library and does no I/O of its own.
package identity
