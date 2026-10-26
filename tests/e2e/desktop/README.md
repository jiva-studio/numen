# Desktop end-to-end tests

`npm test` runs Playwright against the debug server of `scripts/start-server.mjs`: a temporary vault holds the config, cache, data and home folders, the plugins run on mock models with empty placeholder dependencies, and `scripts/stand-ins.mjs` answers the release manifest and the chat API on loopback.

On Linux `scripts/run-offline.mjs` runs the suite inside `unshare -rn` with loopback alone, so a scenario that reaches the internet or production fails; give it a local stand-in instead.
