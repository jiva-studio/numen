# Numen Agent

Built-in AI assistant that chats through a connection to a cloud service (OpenRouter, OpenAI) or to a server on your computer (Ollama, LM Studio).

<!-- more -->

## What You Can Do

- **Chat and research immediately**: Add a connection in Settings that supports chat, add a connection of the type Numen agent, and choose its model in the agent chat.
- **Pick the model**: The models of a Numen agent are the models of every connection that supports chat, listed as "OpenRouter · Claude Sonnet 5.5". The agent chat and Flashcards Studio offer them in the row of the agent. Without a chosen model the agent answers "Choose a model."
- **Full vault access**: Can search your notes and books, follow links, and suggest edits in your workspace, with the tools of the other plugins that its permissions allow.

## Settings

- **Parallel requests** (on the Numen agent connection): How many tasks of the connection run at once (1 by default).
- **Temperature**: How creative or focused the replies should be (`0.0` is more focused, `1.0` is more creative).
- **Max tokens**: The maximum length of each reply.
- **Max steps**: How many requests to the model one task may make before it is stopped (25 by default).
