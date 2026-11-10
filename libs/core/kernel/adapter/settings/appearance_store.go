package settings

import themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"

// AppearanceStore keeps the appearance where the callbacks keep it and passes the warnings on. The
// settings file is read and written by the assembly, which hands its operations in.
type AppearanceStore struct {
	read  func() (themedomain.Appearance, error)
	write func(themedomain.Appearance) error
	warn  func(string)
}

// NewAppearanceStore adapts the read and write callbacks and the warning callback, which may be nil.
func NewAppearanceStore(
	read func() (themedomain.Appearance, error),
	write func(themedomain.Appearance) error,
	warn func(string),
) AppearanceStore {
	return AppearanceStore{read: read, write: write, warn: warn}
}

// ReadAppearance returns the appearance the read callback gives.
func (s AppearanceStore) ReadAppearance() (themedomain.Appearance, error) { return s.read() }

// WriteAppearance passes the appearance to the write callback.
func (s AppearanceStore) WriteAppearance(selected themedomain.Appearance) error {
	return s.write(selected)
}

// WarnTheme passes the text to the warning callback, if there is one.
func (s AppearanceStore) WarnTheme(text string) {
	if s.warn != nil {
		s.warn(text)
	}
}
