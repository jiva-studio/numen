package settings

import "github.com/jiva-studio/numen/libs/core/kernel/domain/logs"

type Logging struct {
	Level string `json:"level"`
}

// LogLevel returns the application logging threshold.
func (f *File) LogLevel() logs.Level {
	if f == nil {
		return logs.DefaultLevel
	}
	return logs.ResolveLevel(f.Texts[keyLogLevel])
}
