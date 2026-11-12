// Package acp is the client side of the Agent Client Protocol: it opens a session with an agent
// program over the program's standard streams, sets the model and the reasoning level, sends prompt
// turns and reports the session updates of each turn. The client offers the agent no file system and
// no terminal, and offers the MCP servers it is given.
package acp
