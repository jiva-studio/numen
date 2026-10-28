# OpenAI Codex Agent

Runs the OpenAI Codex CLI (`codex`) as a local agent provider for vault queries, note editing, and completions.

<!-- more -->

## How It Works

- **Local CLI integration**: Spawns the `codex` CLI binary and connects it to Numen's vault MCP tools via `numen.v1.AgentService`.
- **Models**: Uses the OpenAI reasoning and coding models available in your local Codex CLI configuration.
- **Settings**:
  - **Executable path**: Optional custom path to the `codex` binary. When empty, `codex` is resolved from your system `PATH`.
