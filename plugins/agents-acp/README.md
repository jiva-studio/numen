# ACP Agents

Connects Numen to external AI coding agents that speak the Agent Client Protocol (ACP) and routes assistant tasks through them.

<!-- more -->

## How It Works

- **Agent discovery & delegation**: Launches and communicates with external ACP-compatible agent processes to serve `numen.v1.AgentService`.
- **Vault MCP tools**: Exposes the active run's vault and window tools to each ACP session over an ephemeral HTTP MCP server, scoped strictly to the tools authorized for that turn.
- **Step streaming**: Reports tool invocations, arguments, and outputs back to the Agent Chat view as structured turn steps.
- **On-demand lifecycle**: Started automatically by the host when a conversation or task selects an ACP agent.
