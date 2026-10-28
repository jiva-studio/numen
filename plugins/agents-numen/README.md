# Numen Agent

Built-in AI agent that connects directly to any OpenAI-compatible chat completions API (OpenRouter, OpenAI, Ollama, LM Studio) with full access to vault MCP tools.

<!-- more -->

## How It Works

- **Direct API harness**: Speaks the OpenAI Chat Completions protocol with tool calling, requiring no external CLI installation.
- **Automatic model discovery**: Queries `<Chat API>/models` (for example `https://openrouter.ai/api/v1/models` or `http://localhost:11434/v1/models`) to populate the searchable model list, while also allowing any custom model ID to be entered.
- **Settings**:
  - **Chat API**: Base URL of the OpenAI-compatible service (`https://openrouter.ai/api/v1` by default).
  - **API key**: Stored securely in the system keychain.
  - **Model**: Default model identifier used when a conversation does not specify one.
  - **Temperature & Max Tokens**: Controls sampling randomness (`0.0` to `1.0`) and maximum completion length.
