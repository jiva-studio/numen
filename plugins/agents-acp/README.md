# ACP Agents

Use external AI assistants installed on your computer inside the Agent Chat panel and for AI tasks across your vault.

<!-- more -->

## What You Can Do

- **Chat with your favorite AI tools**: Connect any assistant that supports the Agent Client Protocol so you can ask questions, summarize readings, and draft notes without leaving Numen.
- **Work directly with your vault**: The assistant can search your notes, read passages from books and documents, and propose edits that you can review before accepting.
- **Automatic discovery & multiple agents**: the plugin finds the ACP-compatible assistants installed on your computer (such as `claude-code-acp`, `codex-acp`, `gemini --acp`, `opencode acp`, `cursor-agent acp`, `goose acp`, `kimi acp`, and `qwen --acp`). You can also add multiple custom agent executables in **Executable Paths**.
- **Answer requests of other plugins**: Add a connection of the type ACP agent and choose the program to run. A request from a connection runs in an empty temporary folder, with no tools and without writing to disk.

## Settings

- **Request timeout**: Seconds to wait for the program to answer a request from a connection. Without a value, a request waits 120 seconds.
