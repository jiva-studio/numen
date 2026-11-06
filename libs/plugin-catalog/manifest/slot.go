package manifest

// SlotEntry places one thing of the plugin in a slot of a window, or in a submenu a plugin declares.
// Exactly one of Command, View and Submenu is set.
type SlotEntry struct {
	// Slot is a slot of the list of protocol.Slots, or the id of a submenu.
	Slot string
	// Command is the id the manifest declares the command under, or the full id of a command of
	// another owner, such as app:settings.open. The owner of the slot draws it.
	Command string
	// View is the id the frontend registers the component under. The component draws itself.
	View string
	// Submenu is the id of the submenu the entry declares, which becomes a place of its own.
	Submenu string
	// Title is the title of a submenu.
	Title string
	// Icon is the kebab-case icon name of a submenu; empty for none.
	Icon string
	// Group names the entries a slot draws together; empty is the last group.
	Group string
	// Order sorts the entries of a group, lowest first; nil sorts after every order.
	Order *int
}
