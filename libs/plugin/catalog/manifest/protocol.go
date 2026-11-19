package manifest

// GetProtocol returns the protocol the manifest declares. An omitted protocol is 1.
func (m Manifest) GetProtocol() int {
	return getDeclaredProtocol(m.Protocol)
}

// IsProtocolCompatible reports whether the plugin declaration matches the host protocol.
func IsProtocolCompatible(declared, host int) bool {
	return getDeclaredProtocol(declared) == host
}

func getDeclaredProtocol(declared int) int {
	if declared == 0 {
		return 1
	}
	return declared
}
