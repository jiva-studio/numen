package settings

// Window holds how the window is drawn beyond its appearance. The declaration of the shell gives
// its type and its default.
type Window struct {
	// Buttons is where the window draws its buttons on Linux: `windows` at the right, `mac` at the
	// left.
	Buttons string `json:"buttons"`
}

// DefaultWindow is the window of an unconfigured installation.
func DefaultWindow() Window {
	return Window{Buttons: getAppString(keyWindowButtons)}
}
