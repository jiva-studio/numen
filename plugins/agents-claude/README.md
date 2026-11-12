# Claude Code Agent

Use Anthropic Claude Code (`claude`) installed on your computer to answer questions about your vault, summarize sources, and help write notes.

<!-- more -->

## What You Can Do

- **Research across your vault**: Use Claude Opus, Sonnet, or Haiku through your Claude Code subscription to search notes, books, and transcripts.
- **Guided note editing**: Ask Claude to reorganize notes, extract key ideas, or link related passages, with interactive previews for every edit.
- **Answer requests of other plugins**: Add a connection of the type Claude Code, then choose the model and the level of reasoning in a row. A request from a connection runs the program with no tools and without writing to disk.

## Settings

- **Executable path**: Choose the location of the `claude` program on your computer if it is installed in a custom folder.
- **Hooks & skills**: Allow Claude Code to read your own hooks, skills, standing instructions and plugins. Off by default. What a vault contains is never read.
- **Request timeout**: Seconds to wait for the program to answer a request from a connection. Without a value, a request waits 120 seconds.
