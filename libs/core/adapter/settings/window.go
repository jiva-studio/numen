package settings

// Window holds how the window is drawn beyond its appearance. The declaration of the shell gives
// its type and its default.
type Window struct {
	Buttons string `json:"buttons"`
}

// DefaultWindow is the window of an unconfigured installation.
func DefaultWindow() Window {
	return Window{Buttons: getAppString(keyWindowButtons)}
}
