package transcript

import "encoding/json"

// PartialProgress is how far a partial transcript got: the millisecond of the recording, and the size of
// the partial file at that moment.
type PartialProgress struct {
	MS   int   `json:"ms"`
	Size int64 `json:"size"`
}

// ParseReached reads the bytes Encode wrote.
func ParseReached(raw []byte) (PartialProgress, error) {
	var reached PartialProgress
	if err := json.Unmarshal(raw, &reached); err != nil {
		return PartialProgress{}, err
	}
	return reached, nil
}

// Encode returns the bytes ParseReached reads.
func (r PartialProgress) Encode() ([]byte, error) {
	return json.Marshal(r)
}
