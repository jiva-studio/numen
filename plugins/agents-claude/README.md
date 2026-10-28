# Claude Code Agent

Runs the Anthropic Claude Code CLI (`claude`) as a local agent provider for vault conversations and AI completions.

<!-- more -->

## How It Works

- **Local CLI integration**: Invokes the `claude` CLI process on your machine and bridges its streaming output and MCP tool calls to `numen.v1.AgentService`.
- **Models**: Supports Claude Opus, Sonnet, and Haiku models available to your Claude Code account.
- **Settings**:
  - **Executable path**: Optional custom path to the `claude` binary. When empty, `claude` is resolved from your system `PATH`.
  - **Hooks & skills**: Supports local Claude Code hooks and skills when enabled in Agent settings.
