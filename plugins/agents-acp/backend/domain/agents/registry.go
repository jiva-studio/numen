package agents

// Known is an agent program of the ACP Registry that a person installs: Name is what the registry calls it,
// Program the command on the path, and Arguments the arguments that start its ACP server.
type Known struct {
	Name      string
	Program   string
	Arguments []string
}

// acpFlag is the flag that starts the ACP server of several agents.
const acpFlag = "--acp"

// ListKnown returns the agents of the ACP Registry (github.com/agentclientprotocol/registry, registry.json, read
// on 2026-11-13) that install a program of their own, in the order of their names. The registry also lists
// adapters that run only through npx or uvx downloads, which are left out, and Claude, Codex and Antigravity,
// which have plugins of their own.
func ListKnown() []Known {
	return []Known{
		{Name: "Amp", Program: "amp-acp", Arguments: nil},
		{Name: "Auggie CLI", Program: "auggie", Arguments: []string{acpFlag}},
		{Name: "Autohand Code", Program: "autohand", Arguments: []string{acpFlag}},
		{Name: "Bob", Program: "bob", Arguments: []string{acpSubcommand}},
		{Name: "Cline", Program: "cline", Arguments: []string{acpFlag}},
		{Name: "Codebuddy Code", Program: "codebuddy", Arguments: []string{acpFlag}},
		{Name: "Cortex Code", Program: "cortex", Arguments: []string{acpSubcommand, "serve"}},
		{Name: "Corust Agent", Program: "corust-agent-acp", Arguments: nil},
		{Name: "crow-cli", Program: "crow-cli", Arguments: []string{acpSubcommand}},
		{Name: "Cursor", Program: "cursor-agent", Arguments: []string{acpSubcommand}},
		{Name: "Devin", Program: "devin", Arguments: []string{acpSubcommand}},
		{Name: "DimCode", Program: "dimcode", Arguments: []string{acpSubcommand}},
		{Name: "Dirac", Program: "dirac", Arguments: []string{acpFlag}},
		{Name: "Factory Droid", Program: "droid", Arguments: []string{"exec", "--output-format", "acp-daemon"}},
		{Name: "Gemini CLI", Program: "gemini", Arguments: []string{acpFlag}},
		{Name: "GitHub Copilot", Program: "copilot", Arguments: []string{acpFlag}},
		{Name: "goose", Program: "goose", Arguments: []string{acpSubcommand}},
		{Name: "Grok Build", Program: "grok", Arguments: []string{"agent", "stdio"}},
		{Name: "Harn", Program: "harn", Arguments: []string{"serve", acpSubcommand}},
		{Name: "Junie", Program: "junie", Arguments: []string{"--acp=true"}},
		{Name: "Kilo", Program: "kilo", Arguments: []string{acpSubcommand}},
		{Name: "Kimchi", Program: "kimchi", Arguments: []string{"--mode", acpSubcommand}},
		{Name: "Kimi Code", Program: "kimi", Arguments: []string{acpSubcommand}},
		{Name: "Kiro CLI", Program: "kiro-cli", Arguments: []string{acpSubcommand, "--agent-engine=v3"}},
		{Name: "MiniMax Code", Program: "mcode", Arguments: []string{acpSubcommand}},
		{Name: "Mistral Vibe", Program: "vibe-acp", Arguments: nil},
		{Name: "Nova", Program: "nova", Arguments: []string{acpSubcommand}},
		{Name: "OpenCode", Program: "opencode", Arguments: []string{acpSubcommand}},
		{Name: "Qoder CLI", Program: "qodercli", Arguments: []string{acpFlag}},
		{Name: "Qwen Code", Program: "qwen", Arguments: []string{acpFlag, "--experimental-skills"}},
		{Name: "siGit Code", Program: "sigit", Arguments: nil},
		{Name: "Stakpak", Program: "stakpak", Arguments: []string{acpSubcommand}},
		{Name: "VT Code", Program: "vtcode", Arguments: []string{acpSubcommand}},
	}
}
