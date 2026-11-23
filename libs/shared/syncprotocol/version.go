package syncprotocol

// Version is the protocol version this package implements. A server implements one version.
const Version uint32 = 2

// HeaderVersion is the header every request carries the protocol version in.
const HeaderVersion = "Numen-Sync-Protocol"

// SignInPluginID is the plugin id a sign-in message is signed under.
const SignInPluginID = "md.numen.tools.sync"
