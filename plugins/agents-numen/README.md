# Numen Agent

Built-in AI assistant that chats through a connection to a cloud service (OpenRouter, OpenAI) or to a server on your computer (Ollama, LM Studio).

<!-- more -->

## What You Can Do

- **Chat and research immediately**: Add a connection in Settings, choose it in the Connection setting of this plugin, add a connection of the type Numen agent, choose it in the Agents setting of the agent chat, and start chatting with your vault.
- **Pick the model**: The row of the setting holds the connection, the model and the reasoning level. The models of the connection are listed, and you can type any model name yourself.
- **Full vault access**: Can search your notes and books, follow links, and suggest edits in your workspace, with the tools of the other plugins that its permissions allow.

## Settings

- **Parallel requests** (on the Numen agent connection): How many tasks of the connection run at once (1 by default).
- **Connection**: One row with the connection to chat through, the model and the reasoning level. Only connections that support chat are offered. The address and the key belong to the connection.
- **Temperature**: How creative or focused the replies should be (`0.0` is more focused, `1.0` is more creative).
- **Max tokens**: The maximum length of each reply.
- **Max steps**: How many requests to the model one task may make before it is stopped (25 by default).
