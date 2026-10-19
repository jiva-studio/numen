// Package harness is numen's own agent: a loop over an OpenAI-compatible chat API that offers the
// model the vault's tools, served over MCP, as functions, calls each tool the model asks for and
// reports the work as AgentService steps.
//
// It starts no process and touches nothing of the machine but the network, so it links into a
// plugin's process and into any application that hosts a vault, on every platform Go builds for.
package harness
