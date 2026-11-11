package transcript

import "encoding/json"

// Reached is how far a partial transcript got: the millisecond of the recording, and the size of
// the partial file at that moment.
type Reached struct {
	MS   int   `json:"ms"`
	Size int64 `json:"size"`
}

// ParseReached reads the bytes Encode wrote.
func ParseReached(raw []byte) (Reached, error) {
	var reached Reached
	if err := json.Unmarshal(raw, &reached); err != nil {
		return Reached{}, err
	}
	return reached, nil
}

// Encode returns the bytes ParseReached reads.
func (r Reached) Encode() ([]byte, error) {
	return json.Marshal(r)
}
