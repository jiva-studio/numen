# Antigravity Agent

Runs the Google Antigravity CLI (`agy`) as a local agent provider for the Agent Chat panel and background AI tasks.

<!-- more -->

## How It Works

- **Local CLI integration**: Spawns the `agy` CLI installed on your machine and streams responses and tool calls through `numen.v1.AgentService`.
- **Models**: Supports Gemini 2.5 Pro, Gemini 2.5 Flash, and other models configured in your local Antigravity installation.
- **Settings**:
  - **Executable path**: Optional custom path to the `agy` binary. When empty, `agy` is resolved from your system `PATH`.
  - **Hooks & skills**: Can load local Antigravity skills and hooks when enabled in Agent settings.
