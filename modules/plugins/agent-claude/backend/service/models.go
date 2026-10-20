package service

import "github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/reasoning"

// AgentModel is one model the program can be asked for.
type AgentModel struct {
	Name  string
	Title string
}

// claudeModels are the aliases the program accepts for --model. They are the same on every
// installation and need no account to be listed.
var claudeModels = []AgentModel{
	{Name: "sonnet", Title: "Sonnet"},
	{Name: "opus", Title: "Opus"},
	{Name: "haiku", Title: "Haiku"},
	{Name: "opusplan", Title: "Opus plan, Sonnet execution"},
}

// claudeReasoning are the depths the program accepts for --effort, which every model of claudeModels
// works at; the program runs a level the model lacks at the highest one below it.
var claudeReasoning = reasoning.NewLevels("low", "medium", "high", "xhigh", "max")
