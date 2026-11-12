# ACP Agents

Use the AI assistants installed on your computer that implement the Agent Client Protocol, such as Gemini CLI, OpenCode, Goose, Cursor Agent, Kimi and Qwen Code, in the Agent Chat panel and for AI tasks across your vault.

<!-- more -->

## What You Can Do

- **Chat with your favorite AI tools**: Add a connection of the type ACP agent, choose its program, and choose the connection as an agent of the chat. The assistant answers in the chat as it writes.
- **Work directly with your vault**: The assistant can call the tools of your vault, such as searching your notes and reading passages from books and documents. It may call these tools and no others: it reads no file of your computer and runs no command.
- **Continue a conversation**: The turns of one chat continue one session of the assistant. A session unused for 30 minutes is closed, and at most 8 are kept.
- **Any ACP assistant**: The connection runs any program that implements the Agent Client Protocol, such as `gemini --acp`, `opencode acp`, `cursor-agent acp`, `goose acp`, `kimi acp` and `qwen --acp`.
- **Answer requests of other plugins**: A request of another plugin runs in an empty temporary folder, with no tools and without writing to disk.

Claude Code, Codex and Antigravity have plugins of their own: Claude Code Agent, OpenAI Codex Agent and Antigravity Agent.

## Settings

- **Request timeout**: Seconds to wait for the program to answer a request from a connection, or to open the session of a task. Without a value, it waits 120 seconds.
