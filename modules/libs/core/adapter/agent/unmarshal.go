package agent

import "encoding/json"

// unmarshal is json.Unmarshal, named here so the shape a file is read through is
// one function.
func unmarshal(raw []byte, into any) error { return json.Unmarshal(raw, into) }
