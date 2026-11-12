package model

// ConnectionProfile is what the connection that answers for the plugin says about its model. The
// prefixes are part of what a vector is, so they are part of the handling.
type ConnectionProfile struct {
	Model          string
	Source         string
	Pooling        string
	QueryPrefix    string
	DocumentPrefix string
	Dimensions     int
	MaxTokens      int
}

// GetHandling returns what is done to a text before the connection reads it.
func (p ConnectionProfile) GetHandling() Handling {
	return Handling{
		QueryPrefix: p.QueryPrefix, DocumentPrefix: p.DocumentPrefix, File: "", Normalize: false, ExtraInputs: nil,
	}
}

// GetHandlingHash returns the hash of the handling, empty when the handling changes nothing.
func (p ConnectionProfile) GetHandlingHash() string {
	handling := p.GetHandling()
	if handling.IsZero() {
		return ""
	}
	return handling.Hash(p.MaxTokens)
}
