package agent

import "context"

// antigravityModels is the Antigravity CLI asked which models it offers.
type antigravityModels struct{}

// GetModels reports none: `agy models` prints a list for people, and its layout is not documented.
func (antigravityModels) GetModels(context.Context) ([]AgentModel, error) { return nil, nil }
