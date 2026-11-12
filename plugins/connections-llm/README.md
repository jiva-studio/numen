# Language model services

Connects Numen to OpenRouter, OpenAI and servers that speak the OpenAI API, such as Ollama or LM Studio, so that plugins can use their models for completions and embeddings.

<!-- more -->

## What You Can Do

- **Add a service as a connection**: Pick OpenRouter, OpenAI or an OpenAI-compatible server when you add a connection. The key is kept in the system keychain and is never shown to the plugins that use the connection.
- **Use your own server**: For an OpenAI-compatible server, enter its address, for example `http://localhost:11434/v1`. A key is optional. A key is sent over `http` only to this computer, and over `https` to any address.
- **Choose a model for each use**: The models the service lists are offered where you choose a model. You can also type a model name the service does not list. The list is kept for a day, also across restarts, and is listed again in the background after that.
- **Set reasoning**: OpenRouter and the reasoning models of OpenAI take a level of low, medium or high.

## Settings

- **Request timeout**: How many seconds to wait for a service to answer. Without a value a completion waits 120 seconds and an embedding 60 seconds. The first request to a model of an OpenAI-compatible server waits up to 10 minutes, because a local server loads the model on it.
