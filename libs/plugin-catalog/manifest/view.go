package manifest

import (
	"fmt"
)

// SlotTitleBar is the slot at the top of a window. The view in it draws the whole title bar.
const SlotTitleBar = "titleBar"

// isSlot reports whether the name is a place of a window a view may be drawn in.
func isSlot(name string) bool { return name == SlotTitleBar }

// View is a component of the plugin's frontend that the window draws in one of its slots.
type View struct {
	// ID is the id the frontend registers the component under.
	ID string
	// Slot is the name of the place in the window, such as SlotTitleBar.
	Slot string
}

// listViewIssues checks that each view has an id written as the frontend registers it, that no id is
// declared twice, that each names a slot of the window, and that no slot is declared twice.
func listViewIssues(views []View) []error {
	var issues []error
	observedIDs := map[string]bool{}
	observedSlots := map[string]bool{}
	for _, declared := range views {
		switch {
		case !tabKindPattern.MatchString(declared.ID):
			issues = append(issues, fmt.Errorf("%w: view %q is not lowercase words", ErrInvalidContribution, declared.ID))
		case observedIDs[declared.ID]:
			issues = append(issues, fmt.Errorf("%w: view %q is declared twice", ErrInvalidContribution, declared.ID))
		}
		observedIDs[declared.ID] = true
		switch {
		case !isSlot(declared.Slot):
			issues = append(issues, fmt.Errorf("%w: view %q names slot %q, which the window does not have", ErrInvalidContribution, declared.ID, declared.Slot))
		case observedSlots[declared.Slot]:
			issues = append(issues, fmt.Errorf("%w: slot %q is declared by two views", ErrInvalidContribution, declared.Slot))
		}
		observedSlots[declared.Slot] = true
	}
	return issues
}
