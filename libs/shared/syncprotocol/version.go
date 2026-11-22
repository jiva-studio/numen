package syncprotocol

// Version is the protocol version this package speaks. A server speaks one version.
const Version uint32 = 1

// HeaderVersion is the header every request carries the protocol version in.
const HeaderVersion = "Numen-Sync-Protocol"
