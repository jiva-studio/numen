package fetch

// Config lists the programs that this machine uses to access a video site.
type Config struct {
	Video Tool
	Sound Tool
}

// Tool is a program this machine holds.
type Tool struct {
	Command     []string
	Arguments   []string
	Environment map[string]string
}

// Defaults is the programs PATH and the host name.
func Defaults() Config {
	// gomutants:disable-next-line RETURN_ZERO reason="explicit nil fields equal zero Config"
	return Config{
		Video: Tool{Command: nil, Arguments: nil, Environment: nil},
		Sound: Tool{Command: nil, Arguments: nil, Environment: nil},
	}
}
