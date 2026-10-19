package epub

// Resolve resolves an href against a base directory.
func Resolve(base, href string) string { return resolve(base, href) }

// HasTokenBytes reports whether a space-separated attribute carries a value.
func HasTokenBytes(attribute []byte, token string) bool { return hasTokenBytes(attribute, token) }

// ResolveHref resolves an href against a base directory, keeping the fragment.
func ResolveHref(base, href string) string { return resolveHref(base, href) }
