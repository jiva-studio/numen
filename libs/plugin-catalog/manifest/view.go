package manifest

import (
	"fmt"
	"slices"
)

// SlotTitleBar is the slot at the top of a window. The view in it draws the whole title bar.
const SlotTitleBar = "titleBar"

// slots are the places of a window a view may be drawn in.
var slots = []string{SlotTitleBar}

// View is a component of the plugin's frontend that the window draws in one of its slots.
type View struct {
	// ID is the id the frontend registers the component under.
	ID string
	// Slot is the name of the place in the window, such as SlotTitleBar.
	Slot string
}

// GetSlotView returns the view the plugin declares for the slot.
func (c Contributes) GetSlotView(slot string) (View, bool) {
	for _, declared := range c.Views {
		if declared.Slot == slot {
			return declared, true
		}
	}
	return View{}, false
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
		case !slices.Contains(slots, declared.Slot):
			issues = append(issues, fmt.Errorf("%w: view %q names slot %q, which the window does not have", ErrInvalidContribution, declared.ID, declared.Slot))
		case observedSlots[declared.Slot]:
			issues = append(issues, fmt.Errorf("%w: slot %q is declared by two views", ErrInvalidContribution, declared.Slot))
		}
		observedSlots[declared.Slot] = true
	}
	return issues
}
