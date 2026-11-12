// Package acp is the client side of the Agent Client Protocol: it opens a session with an agent
// program over the program's standard streams, sets the model, asks a question and reports the
// tool calls of the agent. The client offers the agent no file system and no terminal, and offers
// the tools of the run as its one MCP server.
package acp
