# ACP Agents Plugin

Generic Agent Client Protocol client that discovers programs speaking ACP and answers completions through them.

## Features

- Discovers and connects to external ACP agent processes.
- Serves `numen.v1.AgentService` to delegate completions to ACP-compatible agents.
- Activates on `onCommand:askAgent`.
