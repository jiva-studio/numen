# MCP Server Plugin

Standalone Model Context Protocol (MCP) server plugin for Numén.

It exposes an external MCP SSE, Streamable HTTP, and stdio server that aggregates the tools of the plugins declaring the `tools` capability. It reaches them through `numen.v1.PluginToolService` on the host, which runs each call in the vault the host shows, with no context.
