package subscription

// WarningKey is the key of the health item that tells the subscription does not allow changes, or will not for long.
const WarningKey = "subscription-needed"

const (
	needsText  = "Sync needs a subscription"
	stayText   = "Subscription ended: changes stay on this device"
	graceHead  = "Subscription ended: changes are sent until "
	dateLayout = "2 Jan 2006"
)

// GetWarningText is what the health item says in the phase of the state. An active subscription has none.
func GetWarningText(state State) string {
	switch state.Phase {
	case PhaseActive:
		return ""
	case PhaseGrace:
		return graceHead + state.WritesEnd.Format(dateLayout)
	case PhaseReadOnly, PhaseDeleted:
		return stayText
	case PhaseNone:
	}
	return needsText
}
