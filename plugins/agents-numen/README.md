# Numen Agent

Built-in AI assistant that connects directly to cloud services (OpenRouter, OpenAI) or local models on your computer (Ollama, LM Studio) without needing any extra software installed.

<!-- more -->

## What You Can Do

- **Chat and research immediately**: Enter an API key for OpenRouter or OpenAI—or point to a local model server like Ollama—and start chatting with your vault right away.
- **Browse available models**: Automatically loads the list of models from your chosen service so you can search and pick the one you want, or type any model name yourself.
- **Full vault access**: Can search your notes and books, follow links, and suggest edits in your workspace.

## Settings

- **Chat API**: The web address of your AI service (`https://openrouter.ai/api/v1` by default, or a local address such as `http://localhost:11434/v1` for Ollama).
- **API key**: Your personal access key for the service, kept safely in your system keychain.
- **Model**: The default model to use for new conversations and background tasks.
- **Temperature**: How creative or focused the replies should be (`0.0` is more focused, `1.0` is more creative).
- **Max tokens**: The maximum length of each reply.
