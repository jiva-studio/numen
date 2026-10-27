# ACP Agents Plugin

Generic Agent Client Protocol client that discovers programs speaking ACP and answers completions through them.

## Features

- Discovers and connects to external ACP agent processes.
- Serves `numen.v1.AgentService` to delegate completions to ACP-compatible agents.
- Gives each session the tools of its run as an MCP server over HTTP, and allows the agent to call those tools and nothing else, told by the programmatic name ACP gives a tool call.
- Reports the tool calls of the agent as steps of the run.
- Started by the host when a task is routed to it.
