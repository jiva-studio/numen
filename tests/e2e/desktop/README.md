# Desktop end-to-end tests

`npm test` (and `npm run test:flashcards`) runs Playwright against the debug server of `scripts/start-server.mjs` (`scripts/start-flashcards-server.mjs`): a temporary vault holds the config, cache, data and home folders, the plugins run on mock models with empty placeholder dependencies, and `scripts/stand-ins.mjs` answers the release manifest and the chat API on loopback.

The servers are the debug builds of the two windows, `apps/desktop/numen-debug` and `apps/desktop/numen-flashcards-debug`, built with the `numen_debug` tag (`go build -tags numen_debug -o numen-debug ./cmd/numen` in `apps/desktop`, and the same for `./cmd/numen-flashcards`); `make app_desktop_e2e` builds both.

On Linux `scripts/run-offline.mjs` runs the suite inside `unshare -rn` with loopback alone, so a scenario that reaches the internet or production fails; give it a local stand-in instead.
