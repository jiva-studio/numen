package service

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
