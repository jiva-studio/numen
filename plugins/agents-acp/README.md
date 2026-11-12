# ACP Agents

Use the AI assistants installed on your computer, such as Claude Code, Codex and Antigravity, in the Agent Chat panel and for AI tasks across your vault.

<!-- more -->

## What You Can Do

- **Chat with your favorite AI tools**: Add a connection of the type Claude Code, OpenAI Codex, Antigravity or ACP agent, and choose it as an agent of the chat. The assistant answers in the chat as it writes.
- **Work directly with your vault**: The assistant can call the tools of your vault, such as searching your notes and reading passages from books and documents. It may call these tools and no others: it reads no file of your computer, runs no command, and of its own tools Claude Code keeps web search alone.
- **Continue a conversation**: The turns of one chat continue one session of the assistant. A session unused for 30 minutes is closed, and at most 8 are kept.
- **Any ACP assistant**: The type ACP agent runs any program that speaks the Agent Client Protocol, such as `gemini --acp`, `opencode acp`, `cursor-agent acp`, `goose acp`, `kimi acp` and `qwen --acp`. Choose the program of the connection.
- **Answer requests of other plugins**: A request of another plugin runs in an empty temporary folder, with no tools and without writing to disk.

## Programs

Claude Code, Codex and Antigravity speak ACP through an adapter. Leave the program of the connection empty to use the one on your PATH:

- Claude Code: `claude-agent-acp` (npm `@agentclientprotocol/claude-agent-acp`) or `claude-code-acp`.
- Codex: `codex-acp`.
- Antigravity: `agy`, started as `agy acp`.

## Settings

- **Request timeout**: Seconds to wait for the program to answer a request from a connection, or to open the session of a task. Without a value, it waits 120 seconds.
