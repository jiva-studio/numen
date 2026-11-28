package vault

// CheckWrite allows a write to an unmanaged vault, or by the plugin that manages it. Any other caller,
// the window included (caller empty), gets ErrReadOnly.
func CheckWrite(managedBy, caller string) error {
	if managedBy == "" || managedBy == caller {
		return nil
	}
	return ErrReadOnly
}
