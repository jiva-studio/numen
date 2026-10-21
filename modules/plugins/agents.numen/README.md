# Numen Agent Plugin

Numen's own agent: a harness over an OpenAI-compatible chat API (OpenRouter, OpenAI, Ollama, LM Studio) that works with the vault's tools over MCP.

## Features

- Connects to OpenAI-compatible endpoints to serve agent completions.
- Provides `numen.v1.AgentService` to answer queries and tasks using vault tools over Model Context Protocol (MCP).
- Configurable base URL and model via plugin settings.
