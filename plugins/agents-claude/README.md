# Claude Code Agent

Use Anthropic Claude Code (`claude`) installed on your computer to answer questions about your vault, summarize sources, and help write notes.

<!-- more -->

## What You Can Do

- **Chat in the Agent Chat panel**: When Claude Code is installed on this computer, the connection Claude Code is on the Connections page, marked Discovered, with no setup, and Claude answers in Agent Chat as it writes. A row of the Agents setting of Agent Chat chooses its model and level of reasoning; with no model, the program uses its own default.
- **Research across your vault**: Use Claude Opus, Sonnet, or Haiku through your Claude Code subscription to search notes, books, and transcripts. Claude Code may call the tools of your vault and the web search; it reads no file of your computer and runs no command.
- **Continue a conversation**: The turns of one chat continue one session of Claude Code.
- **Answer requests of other plugins**: A request from a connection runs the program with no tools and without writing to disk.

## Settings

- **Executable path**: Choose the location of the `claude` program on your computer if it is installed in a custom folder.
- **Hooks & skills**: Allow Claude Code to read your own hooks, skills, standing instructions and plugins. Off by default. What a vault contains is never read.
- **Request timeout**: Seconds to wait for the program to answer a request from a connection. Without a value, a request waits 120 seconds.
