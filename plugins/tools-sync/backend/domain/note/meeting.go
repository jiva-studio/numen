package note

// Meeting is how a note of a folder and a note of a synced vault at one path stand to each other.
type Meeting int

const (
	// Same means both notes have the same text.
	Same Meeting = iota + 1
	// LocalOnly means only the folder has a note at the path.
	LocalOnly
	// RemoteOnly means only the synced vault has a note at the path.
	RemoteOnly
	// Differ means both have a note at the path and the texts differ.
	Differ
)

// GetMeeting sorts a pair of notes by the hash of the local note and the hash of the head, each empty when absent.
func GetMeeting(local, head string) Meeting {
	switch {
	case local == head:
		return Same
	case head == "":
		return LocalOnly
	case local == "":
		return RemoteOnly
	}
	return Differ
}
