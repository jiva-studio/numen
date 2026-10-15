package main

import "context"

// GetAntigravityModels reports models supported by Antigravity CLI.
func GetAntigravityModels(_ context.Context) ([]AgentModel, error) {
	return []AgentModel{
		{Name: "gemini-2.5-pro", Title: "Gemini 2.5 Pro", IsDefault: true},
		{Name: "gemini-2.5-flash", Title: "Gemini 2.5 Flash"},
	}, nil
}
