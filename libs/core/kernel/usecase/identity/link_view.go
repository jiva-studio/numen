package identity

import (
	"fmt"

	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

func (l *pendingLink) setOwnIdentity(state identitylib.State, name string) {
	l.progress.DeviceName = name
	l.progress.DeviceFingerprint = state.Record.Device.Public().Fingerprint().String()
	l.progress.AccountFingerprint = state.Chain.Root().Fingerprint().String()
}

func formatCode(value uint32) string { return fmt.Sprintf("%0*d", codeDigits, value) }
