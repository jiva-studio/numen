# ACP Agents

Use the AI assistants installed on your computer that implement the Agent Client Protocol, such as Gemini CLI, OpenCode, Goose, Cursor Agent, Kimi and Qwen Code, in the Agent Chat panel and for AI tasks across your vault.

<!-- more -->

## What You Can Do

- **Chat with the assistants you have installed**: Each assistant of the list below that is installed on this computer is on the Connections page, marked Discovered, with no setup. Another program is added as a connection of the type ACP agent with its program. Choose the connection as an agent of the chat; the assistant answers in the chat as it writes. A row of the Agents setting of Agent Chat chooses its model and level of reasoning; with no model, the program uses the model of its own configuration.
- **Work directly with your vault**: The assistant can call the tools of your vault, such as searching your notes and reading passages from books and documents. It may call these tools and no others: it reads no file of your computer and runs no command.
- **Continue a conversation**: The turns of one chat continue one session of the assistant. A session unused for 30 minutes is closed, and at most 8 are kept.
- **Any ACP assistant**: The connection runs any program that implements the Agent Client Protocol, such as `gemini --acp`, `opencode acp`, `cursor-agent acp`, `goose acp`, `kimi acp` and `qwen --acp --experimental-skills`.
- **Answer requests of other plugins**: A request of another plugin runs in an empty temporary folder, with no tools and without writing to disk.

Claude Code, Codex and Antigravity have plugins of their own: Claude Code Agent, OpenAI Codex Agent and Antigravity Agent.

## Settings

- **Request timeout**: Seconds to wait for the program to answer a request from a connection, or to open the session of a task. Without a value, it waits 120 seconds.

## Assistants Found on This Computer

The list comes from the [ACP Registry](https://agentclientprotocol.com/registry) ([registry.json](https://github.com/agentclientprotocol/registry)) as read on 2026-11-13, and is part of the plugin: nothing is downloaded. An assistant is found when its program is on the path: Amp (`amp-acp`), Auggie CLI (`auggie --acp`), Autohand Code (`autohand --acp`), Bob (`bob acp`), Cline (`cline --acp`), Codebuddy Code (`codebuddy --acp`), Cortex Code (`cortex acp serve`), Corust Agent (`corust-agent-acp`), crow-cli (`crow-cli acp`), Cursor (`cursor-agent acp`), Devin (`devin acp`), DimCode (`dimcode acp`), Dirac (`dirac --acp`), Factory Droid (`droid exec --output-format acp-daemon`), Gemini CLI (`gemini --acp`), GitHub Copilot (`copilot --acp`), goose (`goose acp`), Grok Build (`grok agent stdio`), Harn (`harn serve acp`), Junie (`junie --acp=true`), Kilo (`kilo acp`), Kimchi (`kimchi --mode acp`), Kimi Code (`kimi acp`), Kiro CLI (`kiro-cli acp --agent-engine=v3`), MiniMax Code (`mcode acp`), Mistral Vibe (`vibe-acp`), Nova (`nova acp`), OpenCode (`opencode acp`), Qoder CLI (`qodercli --acp`), Qwen Code (`qwen --acp --experimental-skills`), siGit Code (`sigit`), Stakpak (`stakpak acp`) and VT Code (`vtcode acp`). The registry's adapters that run only through an npx or uvx download are not on the list. A program of one of these names added by hand starts with the same arguments.
