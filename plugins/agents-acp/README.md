# ACP Agents Plugin

Generic Agent Client Protocol client that discovers programs speaking ACP and answers completions through them.

## Features

- Discovers and connects to external ACP agent processes.
- Serves `numen.v1.AgentService` to delegate completions to ACP-compatible agents.
- Gives each session the tools of its run as an MCP server over HTTP, and allows the agent to call those tools and nothing else.
- Activates on `onCommand:askAgent`.
